package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const statePath = ".shoshinvim/state.json"
const manifestPath = ".shoshinvim/manifest.json"
const userPath = "lua/user_config/user.lua"

func Inspect(target string) (Inspection, error) {
	target, err := Canonical(target)
	if err != nil {
		return Inspection{}, err
	}
	out := Inspection{Target: target, Ownership: "NEW"}
	entries, err := os.ReadDir(target)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if len(entries) == 0 {
		return out, nil
	}
	out.Ownership = "READ ONLY"
	mPath, err := safePath(target, manifestPath)
	if err != nil {
		return out, err
	}
	var m Manifest
	if err = DecodeFile(mPath, &m); err != nil {
		if os.IsNotExist(err) {
			journals, jerr := Journals(target)
			if jerr != nil {
				return out, jerr
			}
			for _, j := range journals {
				if j.Status != "committed" && j.Status != "restored" {
					out.Ownership = "RECOVERY"
					out.Problems = append(out.Problems, "Incomplete initial transaction: "+j.ID)
					return out, nil
				}
			}
			if len(journals) > 0 && journals[0].Status == "restored" {
				clean := true
				walkErr := filepath.WalkDir(target, func(path string, d os.DirEntry, e error) error {
					if e != nil {
						return e
					}
					rel, _ := filepath.Rel(target, path)
					if rel == filepath.Join(".shoshinvim", "transactions") {
						return filepath.SkipDir
					}
					if !d.IsDir() {
						clean = false
					}
					return nil
				})
				if walkErr != nil {
					return out, walkErr
				}
				if clean {
					out.Ownership = "NEW"
					return out, nil
				}
			}
		}
		out.Problems = append(out.Problems, "No valid ownership manifest; existing configuration is read only.")
		return out, nil
	}
	if m.Schema != SchemaVersion {
		return out, fmt.Errorf("unsupported manifest version")
	}
	out.Ownership = "SAVED"
	if _, ok := m.Files[statePath]; !ok {
		return out, fmt.Errorf("manifest does not own state")
	}
	for name, expected := range m.Files {
		h, err := fileHash(target, name)
		if err != nil || h != expected {
			out.Problems = append(out.Problems, "Drift: "+name)
			out.Ownership = "RECOVERY"
		}
	}
	sPath, err := safePath(target, statePath)
	if err != nil {
		return out, err
	}
	var s State
	if err = DecodeFile(sPath, &s); err != nil {
		return out, err
	}
	if err = LoadCatalog().Validate(s); err != nil {
		return out, err
	}
	out.State = &s
	journals, err := Journals(target)
	if err != nil {
		return out, err
	}
	for _, j := range journals {
		if j.Status != "committed" && j.Status != "restored" {
			out.Ownership = "RECOVERY"
			out.Problems = append(out.Problems, "Incomplete transaction: "+j.ID)
		}
	}
	return out, nil
}

func targetIdentity(target string) (string, error) {
	// Bind an existing target to filesystem identity; missing targets bind to their nearest existing parent.
	p := target
	for {
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.IsDir() {
				return "", fmt.Errorf("target ancestor is not a directory")
			}
			return identity(fi, p), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		p = filepath.Dir(p)
	}
}

func BuildPlan(ctx context.Context, target string, s State, bin string, external bool) (Plan, error) {
	var p Plan
	ins, err := Inspect(target)
	if err != nil {
		return p, err
	}
	if ins.Ownership == "READ ONLY" || ins.Ownership == "RECOVERY" {
		return p, fmt.Errorf("%s: %s", ins.Ownership, strings.Join(ins.Problems, "; "))
	}
	files, err := Generate(s)
	if err != nil {
		return p, err
	}
	path, version, err := Neovim(ctx, bin)
	if err != nil {
		return p, err
	}
	if err = Probe(ctx, path, files); err != nil {
		return p, err
	}
	id, err := targetIdentity(ins.Target)
	if err != nil {
		return p, err
	}
	p = Plan{Schema: SchemaVersion, Catalog: CatalogVersion, Target: ins.Target, Identity: id, Neovim: path, NeovimVersion: version, State: Clone(s), RunExternal: external}
	var old Manifest
	old.Files = map[string]string{}
	if ins.State != nil {
		mp, _ := safePath(p.Target, manifestPath)
		if err = DecodeFile(mp, &old); err != nil {
			return p, err
		}
	}
	files[statePath] = string(JSON(s))
	manifest := Manifest{Schema: SchemaVersion, Files: map[string]string{}}
	for name, content := range files {
		manifest.Files[name] = Hash([]byte(content))
	}
	files[manifestPath] = string(JSON(manifest))
	for _, name := range sorted(files) {
		before, err := fileHash(p.Target, name)
		if err != nil {
			return p, err
		}
		if before != "" && name != manifestPath {
			if _, owned := old.Files[name]; !owned {
				return p, fmt.Errorf("unmanaged destination: %s", name)
			}
		}
		role := "generated"
		if strings.HasPrefix(name, ".shoshinvim/") {
			role = "metadata"
		}
		p.Files = append(p.Files, File{Path: name, Content: files[name], Before: before, After: Hash([]byte(files[name])), Role: role})
	}
	for _, name := range sorted(old.Files) {
		if _, ok := files[name]; !ok {
			p.Files = append(p.Files, File{Path: name, Before: old.Files[name], Role: "generated", Delete: true})
		}
	}
	before, err := fileHash(p.Target, userPath)
	if err != nil {
		return p, err
	}
	if before == "" {
		content := "-- Your Lua extensions. ShoshinVim never overwrites this file.\n"
		p.Files = append(p.Files, File{Path: userPath, Content: content, After: Hash([]byte(content)), Role: "user"})
	}
	p.Actions, err = Actions(s, ins.Target)
	if err != nil {
		return p, err
	}
	for _, d := range Dependencies(s, ins.Target) {
		// The chosen Neovim was resolved above; a PATH lookup would contradict --nvim.
		if d.Status == "missing" && d.ID != "nvim" {
			p.Warnings = append(p.Warnings, d.Purpose+": missing "+d.ID)
		}
	}
	p.Digest = p.Fingerprint()
	return p, nil
}

// VerifyPlan rebuilds a plan with the Neovim the caller chose (bin, as --nvim; empty
// means PATH) and returns that Neovim's path. The plan's own Neovim path is only
// compared, never executed: a plan file is data, and its digest is not a signature.
func VerifyPlan(ctx context.Context, p Plan, bin string) (string, error) {
	if p.Schema != SchemaVersion || p.Catalog != CatalogVersion || p.Digest != p.Fingerprint() {
		return "", fmt.Errorf("invalid or incompatible plan")
	}
	nvim, _, err := Neovim(ctx, bin)
	if err != nil {
		return "", err
	}
	if nvim != p.Neovim {
		return "", fmt.Errorf("the plan was reviewed with Neovim at %s, but this run uses %s; review a new plan, or choose that Neovim with --nvim", p.Neovim, nvim)
	}
	fresh, err := BuildPlan(ctx, p.Target, p.State, nvim, p.RunExternal)
	if err != nil {
		return "", err
	}
	// Availability may improve independently; it is evidence, not authorization.
	fresh.Warnings = p.Warnings
	fresh.Digest = fresh.Fingerprint()
	if p.Digest != fresh.Digest {
		return "", fmt.Errorf("stale plan: target, files, actions, or Neovim changed; review a new plan")
	}
	return nvim, nil
}

func Summary(p Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Target: %s\nNeovim: %s\nPlan: %s\n", p.Target, p.NeovimVersion, p.Digest)
	for _, f := range p.Files {
		op := "create"
		if f.Before == f.After {
			op = "unchanged"
		} else if f.Delete {
			op = "remove owned"
		} else if f.Before != "" {
			op = "modify"
		}
		fmt.Fprintf(&b, "  %-13s %s\n", op, f.Path)
	}
	fmt.Fprintf(&b, "Backup/journal: %s\n", filepath.Join(p.Target, ".shoshinvim", "transactions", "<transaction-id>", "journal.json"))
	for _, a := range p.Actions {
		fmt.Fprintf(&b, "  [PENDING] %s -> %s\n", a.Description, a.Destination)
	}
	for _, w := range p.Warnings {
		fmt.Fprintf(&b, "  [MISSING] %s\n", w)
	}
	b.WriteString("Local configuration and external acquisition have separate outcomes.\nRollback restores configuration only; it does not uninstall plugins or tools.\n")
	return b.String()
}
