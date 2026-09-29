package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// configHome and dataHome mirror the bases of Neovim's stdpath('config') and stdpath('data').
func configHome() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return base
	}
	if runtime.GOOS == "windows" {
		return os.Getenv("LOCALAPPDATA")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}
func dataHome() string {
	if base := os.Getenv("XDG_DATA_HOME"); base != "" {
		return base
	}
	if runtime.GOOS == "windows" {
		return os.Getenv("LOCALAPPDATA")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}

func DefaultTarget() string {
	if dir := os.Getenv("SHOSHINVIM_TARGET_DIR"); dir != "" {
		return dir
	}
	app := os.Getenv("NVIM_APPNAME")
	if app == "" {
		app = "nvim"
	}
	return filepath.Join(configHome(), app)
}

// AppName returns the NVIM_APPNAME that makes Neovim load target as its configuration,
// or "" when target lies outside the configuration home and must be started with -u.
func AppName(target string) string {
	home, err := resolveExisting(configHome())
	if err != nil {
		return ""
	}
	target, err = resolveExisting(target)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(home, target)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.ToSlash(rel)
}

// DataDirFor is where plugins and tools for target live: the shoshinvim folder in the
// stdpath('data') that Neovim uses when it loads target. The generated configuration
// looks there, so it depends on the target, never on the environment ShoshinVim runs in.
func DataDirFor(target string) string {
	app := AppName(target)
	if app == "" {
		app = "nvim" // `nvim -u <target>/init.lua` runs with the default application name.
	}
	if runtime.GOOS == "windows" {
		app += "-data"
	}
	return filepath.Join(dataHome(), filepath.FromSlash(app), "shoshinvim")
}

func Canonical(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if fi, err := os.Lstat(abs); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("target must not be a symlink: %s", abs)
	}
	return resolveExisting(abs)
}

// resolveExisting resolves symlinks in the longest existing ancestor of path.
func resolveExisting(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	cur := abs
	var tail []string
	for {
		_, err := os.Lstat(cur)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		tail = append(tail, filepath.Base(cur))
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("no existing target ancestor")
		}
		cur = parent
	}
	base, err := filepath.EvalSymlinks(cur)
	if err != nil {
		return "", err
	}
	for i := len(tail) - 1; i >= 0; i-- {
		base = filepath.Join(base, tail[i])
	}
	return base, nil
}
func safePath(target, relative string) (string, error) {
	if filepath.IsAbs(relative) || relative == "" || strings.Contains(relative, "\\") {
		return "", fmt.Errorf("unsafe relative path %q", relative)
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes target")
	}
	if clean == "." {
		return "", fmt.Errorf("path is target directory")
	}
	cur := target
	if fi, err := os.Lstat(cur); err == nil && (!fi.IsDir() || fi.Mode()&os.ModeSymlink != 0) {
		return "", fmt.Errorf("unsafe target %s", cur)
	}
	for _, component := range strings.Split(clean, string(filepath.Separator)) {
		cur = filepath.Join(cur, component)
		fi, err := os.Lstat(cur)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing symlink %s", cur)
		}
	}
	return filepath.Join(target, clean), nil
}
func readPath(target, relative string) ([]byte, error) {
	p, err := safePath(target, relative)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}
func fileHash(target, relative string) (string, error) {
	b, err := readPath(target, relative)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return Hash(b), nil
}
