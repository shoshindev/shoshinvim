package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Backup struct {
	Path    string `json:"path"`
	Content []byte `json:"content,omitempty"`
	Hash    string `json:"sha256"`
	Exists  bool   `json:"exists"`
	Mode    uint32 `json:"mode"`
}
type Journal struct {
	ID      string   `json:"id"`
	Target  string   `json:"target"`
	Status  string   `json:"status"`
	Plan    Plan     `json:"plan"`
	Backups []Backup `json:"backups"`
	Digest  string   `json:"digest"`
}
type ApplyOptions struct {
	Hook     func(string) error
	Progress func(string)
}

func AtomicWrite(path string, b []byte, mode os.FileMode) error {
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlink %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".shoshinvim-tmp-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = replace(tmp, path); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	syncErr := d.Sync()
	closeErr := d.Close()
	// Windows replacement requests write-through; directory fsync is not supported there.
	if runtime.GOOS != "windows" && syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
}
func acquireLock(target string) (func(), error) {
	parent := os.TempDir()
	p := filepath.Join(parent, ".shoshinvim-"+Hash([]byte(target))[:16]+".lock")
	if fi, e := os.Lstat(p); e == nil && fi.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("lock is a symlink")
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = lockFile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("another operation owns this target: %w", err)
	}
	return func() { unlockFile(f); f.Close() }, nil
}
func journalPath(target, id string) (string, error) {
	if strings.ContainsAny(id, "/\\.") || id == "" {
		return "", fmt.Errorf("invalid transaction ID")
	}
	return safePath(target, ".shoshinvim/transactions/"+id+"/journal.json")
}
func saveJournal(j *Journal) error {
	j.Digest = ""
	j.Digest = Hash(JSON(j))
	p, err := journalPath(j.Target, j.ID)
	if err != nil {
		return err
	}
	return AtomicWrite(p, JSON(j), 0600)
}
func loadJournal(target, id string) (Journal, error) {
	var j Journal
	p, err := journalPath(target, id)
	if err != nil {
		return j, err
	}
	if err = DecodeFile(p, &j); err != nil {
		return j, err
	}
	digest := j.Digest
	j.Digest = ""
	if digest != Hash(JSON(j)) || j.Target != target || j.ID != id {
		return j, fmt.Errorf("invalid recovery journal")
	}
	j.Digest = digest
	for _, b := range j.Backups {
		if b.Exists && Hash(b.Content) != b.Hash {
			return j, fmt.Errorf("corrupt backup for %s", b.Path)
		}
	}
	return j, nil
}
func Journals(target string) ([]Journal, error) {
	p, err := safePath(target, ".shoshinvim/transactions")
	if err != nil {
		return nil, err
	}
	dirs, err := os.ReadDir(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Journal
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		j, err := loadJournal(target, d.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

func Apply(ctx context.Context, p Plan, opts ApplyOptions) (ApplyResult, error) {
	var result ApplyResult
	unlock, err := acquireLock(p.Target)
	if err != nil {
		return result, err
	}
	defer unlock()
	if err = VerifyPlan(ctx, p); err != nil {
		return result, err
	}
	if opts.Hook != nil {
		if err = opts.Hook("validated"); err != nil {
			return result, err
		}
	}
	j := Journal{ID: fmt.Sprintf("%d", time.Now().UnixNano()), Target: p.Target, Status: "prepared", Plan: p}
	for _, f := range p.Files {
		if f.Before == f.After {
			continue
		}
		backup := Backup{Path: f.Path, Hash: f.Before, Exists: f.Before != "", Mode: 0600}
		if backup.Exists {
			b, err := readPath(p.Target, f.Path)
			if err != nil {
				return result, err
			}
			if Hash(b) != f.Before {
				return result, fmt.Errorf("file changed before backup: %s", f.Path)
			}
			backup.Content = b
			path, _ := safePath(p.Target, f.Path)
			fi, err := os.Stat(path)
			if err != nil {
				return result, err
			}
			backup.Mode = uint32(fi.Mode().Perm())
		}
		j.Backups = append(j.Backups, backup)
	}
	if err = saveJournal(&j); err != nil {
		return result, err
	}
	result.Transaction = j.ID
	phase := func(name string) error {
		if opts.Progress != nil {
			opts.Progress(name)
		}
		if opts.Hook != nil {
			return opts.Hook(name)
		}
		return nil
	}
	local := func() error {
		if err := phase("backed-up"); err != nil {
			return err
		}
		j.Status = "writing"
		if err := saveJournal(&j); err != nil {
			return err
		}
		writeFiles := func(metadata bool) error {
			for _, f := range p.Files {
				if (f.Role == "metadata") != metadata {
					continue
				}
				if f.Before == f.After {
					continue
				}
				path, err := safePath(p.Target, f.Path)
				if err != nil {
					return err
				}
				current, err := fileHash(p.Target, f.Path)
				if err != nil {
					return err
				}
				if current != f.Before {
					return fmt.Errorf("file changed during save: %s", f.Path)
				}
				if f.Delete {
					err = os.Remove(path)
				} else {
					err = AtomicWrite(path, []byte(f.Content), 0600)
				}
				if err != nil {
					return err
				}
				if err = phase("wrote:" + f.Path); err != nil {
					return err
				}
			}
			return nil
		}
		if err := writeFiles(false); err != nil {
			return err
		}
		j.Status = "verifying"
		if err := saveJournal(&j); err != nil {
			return err
		}
		actual := map[string]string{}
		for _, f := range p.Files {
			if f.Role == "metadata" {
				continue
			}
			h, err := fileHash(p.Target, f.Path)
			if err != nil {
				return err
			}
			if h != f.After {
				return fmt.Errorf("post-write hash mismatch: %s", f.Path)
			}
			if f.Role == "generated" && !f.Delete {
				b, err := readPath(p.Target, f.Path)
				if err != nil {
					return err
				}
				actual[f.Path] = string(b)
			}
		}
		if err := Probe(context.WithoutCancel(ctx), p.Neovim, actual); err != nil {
			return err
		}
		if err := writeFiles(true); err != nil {
			return err
		}
		for _, f := range p.Files {
			if f.Role == "metadata" {
				h, e := fileHash(p.Target, f.Path)
				if e != nil || h != f.After {
					return fmt.Errorf("metadata verification failed: %s", f.Path)
				}
			}
		}
		if err := phase("verified"); err != nil {
			return err
		}
		j.Status = "committed"
		return saveJournal(&j)
	}
	if err = local(); err != nil {
		restoreErr := restore(&j)
		if restoreErr != nil {
			return result, fmt.Errorf("local save failed: %v; RECOVERY REQUIRED: %w", err, restoreErr)
		}
		return result, fmt.Errorf("local save failed; previous files restored: %w", err)
	}
	result.LocalCommitted = true
	if p.RunExternal {
		if opts.Progress != nil {
			opts.Progress("LOCAL_COMMITTED")
		}
		result.External = RunActions(ctx, p.Target, p.Actions, opts.Progress)
	}
	return result, nil
}
func restore(j *Journal) error {
	// Check every path before undoing any file. Concurrent/manual changes block restoration.
	for _, b := range j.Backups {
		current, err := fileHash(j.Target, b.Path)
		if err != nil {
			return err
		}
		after := ""
		for _, f := range j.Plan.Files {
			if f.Path == b.Path {
				after = f.After
			}
		}
		if current != b.Hash && current != after {
			return fmt.Errorf("manual content at %s; preserve it before restoration", b.Path)
		}
	}
	for i := len(j.Backups) - 1; i >= 0; i-- {
		b := j.Backups[i]
		path, err := safePath(j.Target, b.Path)
		if err != nil {
			return err
		}
		if b.Exists {
			err = AtomicWrite(path, b.Content, os.FileMode(b.Mode))
		} else {
			err = os.Remove(path)
			if os.IsNotExist(err) {
				err = nil
			}
		}
		if err != nil {
			return err
		}
	}
	j.Status = "restored"
	return saveJournal(j)
}
func Rollback(target, id string) error {
	target, err := Canonical(target)
	if err != nil {
		return err
	}
	unlock, err := acquireLock(target)
	if err != nil {
		return err
	}
	defer unlock()
	j, err := loadJournal(target, id)
	if err != nil {
		return err
	}
	return restore(&j)
}

// RecoverGenerated archives every changed byte before restoring the latest committed generated version.
// It deliberately never executes or merges those manual Lua fragments.
func RecoverGenerated(target, id string) error {
	target, err := Canonical(target)
	if err != nil {
		return err
	}
	unlock, err := acquireLock(target)
	if err != nil {
		return err
	}
	defer unlock()
	j, err := loadJournal(target, id)
	if err != nil {
		return err
	}
	if j.Status != "committed" {
		return fmt.Errorf("choose a committed transaction")
	}
	archive := fmt.Sprintf(".shoshinvim/recovery/%d", time.Now().UnixNano())
	recovery := Journal{ID: fmt.Sprintf("%d", time.Now().UnixNano()), Target: target, Status: "prepared", Plan: j.Plan}
	recovery.Plan.Files = nil
	for _, f := range j.Plan.Files {
		if f.Role == "user" || f.Delete {
			continue
		}
		if Hash([]byte(f.Content)) != f.After {
			return fmt.Errorf("corrupt generated snapshot")
		}
		current, err := fileHash(target, f.Path)
		if err != nil {
			return err
		}
		entry := f
		entry.Before = current
		recovery.Plan.Files = append(recovery.Plan.Files, entry)
		if current != f.After {
			backup := Backup{Path: f.Path, Exists: current != "", Hash: current, Mode: 0600}
			if backup.Exists {
				backup.Content, err = readPath(target, f.Path)
				if err != nil {
					return err
				}
			}
			recovery.Backups = append(recovery.Backups, backup)
		}
		if current != f.After && current != "" {
			content, err := readPath(target, f.Path)
			if err != nil {
				return err
			}
			path, err := safePath(target, archive+"/"+f.Path)
			if err != nil {
				return err
			}
			if err = AtomicWrite(path, content, 0600); err != nil {
				return err
			}
		}
	}
	if err = saveJournal(&recovery); err != nil {
		return err
	}
	for _, f := range recovery.Plan.Files {
		if f.Role == "user" || f.Delete {
			continue
		}
		path, err := safePath(target, f.Path)
		if err != nil {
			return err
		}
		if current, e := fileHash(target, f.Path); e != nil || current != f.Before {
			return fmt.Errorf("file changed during recovery: %s", f.Path)
		}
		if err = AtomicWrite(path, []byte(f.Content), 0600); err != nil {
			return err
		}
		if h, e := fileHash(target, f.Path); e != nil || h != f.After {
			return fmt.Errorf("recovery verification failed: %s", f.Path)
		}
	}
	recovery.Status = "committed"
	return saveJournal(&recovery)
}
