package distribution

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/shoshindev/shoshinvim/internal/engine"
)

const Repository = "shoshindev/shoshinvim"

type Receipt struct {
	Version       string `json:"version"`
	Source        string `json:"source"`
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	ArchiveSHA256 string `json:"archive_sha256"`
	Timestamp     string `json:"timestamp"`
}

func ReceiptPath() string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "state")
		if runtime.GOOS == "windows" {
			base = os.Getenv("LOCALAPPDATA")
		}
	}
	return filepath.Join(base, "shoshinvim", "install.json")
}
func executable() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(p)
}
func owned(path string) (Receipt, error) {
	var r Receipt
	if err := engine.DecodeFile(ReceiptPath(), &r); err != nil {
		return r, err
	}
	a, err := filepath.EvalSymlinks(r.Path)
	if err != nil {
		return r, err
	}
	if a != path || r.Source != "script" || (filepath.Base(path) != "shoshinvim" && filepath.Base(path) != "shoshinvim.exe") {
		return r, fmt.Errorf("receipt does not own this executable")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	if engine.Hash(b) != r.SHA256 {
		return r, fmt.Errorf("executable differs from install receipt")
	}
	return r, nil
}
func ownerAdvice(path string, out io.Writer) {
	fmt.Fprintf(out, "This executable is manually, source, or package-manager installed: %s\nUse its original installation owner. Releases: https://github.com/%s/releases\nSource update: go install github.com/%s/cmd/shoshinvim@<release-tag>\n", path, Repository, Repository)
}
func Uninstall(confirm func(string) error, out io.Writer) error {
	path, err := executable()
	if err != nil {
		return err
	}
	_, err = owned(path)
	if err != nil {
		ownerAdvice(path, out)
		return nil
	}
	if err = confirm("Remove application " + path + " and receipt " + ReceiptPath() + "? Generated Neovim configurations, backups and tools remain."); err != nil {
		return err
	}
	if err = os.Remove(path); err != nil {
		return fmt.Errorf("could not remove executable; it remains usable. Close other running instances and remove %s: %w", path, err)
	}
	if err = os.Remove(ReceiptPath()); err != nil {
		return err
	}
	fmt.Fprintln(out, "Application removed. Neovim configuration and tools were preserved.")
	return nil
}

var validTag = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?$`)

func fetch(ctx context.Context, url string, limit int64) ([]byte, error) {
	client := &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" || len(via) > 8 {
			return fmt.Errorf("unsafe redirect")
		}
		return nil
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ShoshinVim updater")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("download exceeds size limit")
	}
	return b, err
}
func Asset(tag, goos, arch string) (string, error) {
	triples := map[string]string{"darwin/arm64": "aarch64-apple-darwin", "darwin/amd64": "x86_64-apple-darwin", "linux/arm64": "aarch64-unknown-linux-gnu", "linux/amd64": "x86_64-unknown-linux-gnu", "windows/amd64": "x86_64-pc-windows-msvc"}
	triple, ok := triples[goos+"/"+arch]
	if !ok || !validTag.MatchString(tag) {
		return "", fmt.Errorf("unsupported release target")
	}
	extension := ".tar.gz"
	if goos == "windows" {
		extension = ".zip"
	}
	return "shoshinvim_" + strings.TrimPrefix(tag, "v") + "_" + triple + extension, nil
}
func VerifyChecksum(manifest []byte, name string, data []byte) error {
	count := 0
	expected := ""
	for _, line := range strings.Split(string(manifest), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			count++
			expected = fields[0]
		}
	}
	if count != 1 || expected != engine.Hash(data) {
		return fmt.Errorf("archive checksum verification failed")
	}
	return nil
}
func Extract(data []byte, zipArchive bool) ([]byte, error) {
	name := "shoshinvim"
	var result []byte
	count := 0
	if zipArchive {
		z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, f := range z.File {
			if f.Name == "shoshinvim.exe" {
				if !f.Mode().IsRegular() {
					return nil, fmt.Errorf("invalid executable entry")
				}
				count++
				r, err := f.Open()
				if err != nil {
					return nil, err
				}
				result, err = io.ReadAll(io.LimitReader(r, 128<<20))
				r.Close()
				if err != nil {
					return nil, err
				}
			}
		}
	} else {
		gz, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		tr := tar.NewReader(gz)
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			if h.Name == name {
				if h.Typeflag != tar.TypeReg || h.Size > 128<<20 {
					return nil, fmt.Errorf("invalid executable entry")
				}
				count++
				result, err = io.ReadAll(tr)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if count != 1 || len(result) == 0 {
		return nil, fmt.Errorf("archive must contain one expected executable")
	}
	return result, nil
}
func Update(ctx context.Context, current string, confirm func(string) error, out io.Writer) error {
	path, err := executable()
	if err != nil {
		return err
	}
	receipt, err := owned(path)
	if err != nil {
		ownerAdvice(path, out)
		return nil
	}
	if err = confirm("Check GitHub for the latest stable ShoshinVim release?"); err != nil {
		return err
	}
	b, err := fetch(ctx, "https://api.github.com/repos/"+Repository+"/releases/latest", 1<<20)
	if err != nil {
		return err
	}
	var release struct {
		Tag        string `json:"tag_name"`
		Body       string `json:"body"`
		Prerelease bool   `json:"prerelease"`
	}
	if err = json.Unmarshal(b, &release); err != nil {
		return err
	}
	if !validTag.MatchString(release.Tag) || release.Prerelease {
		return fmt.Errorf("invalid stable release metadata")
	}
	if strings.TrimPrefix(release.Tag, "v") == strings.TrimPrefix(current, "v") {
		fmt.Fprintln(out, "Already at latest stable release.")
		return nil
	}
	if err = confirm("Update " + current + " -> " + release.Tag + "\n" + release.Body); err != nil {
		return err
	}
	name, err := Asset(release.Tag, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	base := "https://github.com/" + Repository + "/releases/download/" + release.Tag + "/"
	manifest, err := fetch(ctx, base+"SHA256SUMS", 1<<20)
	if err != nil {
		return err
	}
	archive, err := fetch(ctx, base+name, 192<<20)
	if err != nil {
		return err
	}
	if err = VerifyChecksum(manifest, name, archive); err != nil {
		return err
	}
	binary, err := Extract(archive, runtime.GOOS == "windows")
	if err != nil {
		return err
	}
	// Recheck install ownership after downloads and before replacement.
	if _, err = owned(path); err != nil {
		return err
	}
	old, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err = engine.AtomicWrite(path, binary, 0755); err != nil {
		return err
	}
	receipt.Version = release.Tag
	receipt.SHA256 = engine.Hash(binary)
	receipt.ArchiveSHA256 = engine.Hash(archive)
	receipt.Timestamp = time.Now().UTC().Format(time.RFC3339)
	if err = engine.AtomicWrite(ReceiptPath(), engine.JSON(receipt), 0600); err != nil {
		restoreErr := engine.AtomicWrite(path, old, 0755)
		return fmt.Errorf("receipt update failed: %v; executable restore: %v", err, restoreErr)
	}
	fmt.Fprintln(out, "Verified update installed: "+release.Tag)
	return nil
}
