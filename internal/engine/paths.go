package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func DefaultTarget() string {
	if dir := os.Getenv("SHOSHINVIM_TARGET_DIR"); dir != "" {
		return dir
	}
	app := os.Getenv("NVIM_APPNAME")
	if app == "" {
		app = "nvim"
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		if runtime.GOOS == "windows" {
			base = os.Getenv("LOCALAPPDATA")
		} else {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(base, app)
}
func DataDir() string {
	app := os.Getenv("NVIM_APPNAME")
	if app == "" {
		app = "nvim"
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		if runtime.GOOS == "windows" {
			return filepath.Join(os.Getenv("LOCALAPPDATA"), app+"-data", "shoshinvim")
		}
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, app, "shoshinvim")
}
func Canonical(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if fi, err := os.Lstat(abs); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("target must not be a symlink: %s", abs)
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
