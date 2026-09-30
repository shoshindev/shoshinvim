package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func isolatedEnv(root string) []string {
	replace := map[string]string{"HOME": root, "USERPROFILE": root, "LOCALAPPDATA": filepath.Join(root, "local"), "XDG_CONFIG_HOME": filepath.Join(root, "config"), "XDG_DATA_HOME": filepath.Join(root, "data"), "XDG_STATE_HOME": filepath.Join(root, "state"), "XDG_CACHE_HOME": filepath.Join(root, "cache"), "NVIM_APPNAME": "nvim", "VIM_LOG_FILE": filepath.Join(root, "nvim.log"), "NVIM_LOG_FILE": filepath.Join(root, "nvim.log"), "NVIM": "", "VIMINIT": "", "EXINIT": ""}
	var env []string
	for _, pair := range os.Environ() {
		key := strings.SplitN(pair, "=", 2)[0]
		if _, ok := replace[key]; !ok {
			env = append(env, pair)
		}
	}
	for k, v := range replace {
		env = append(env, k+"="+v)
	}
	return env
}

func Neovim(ctx context.Context, bin string) (string, string, error) {
	if bin == "" {
		bin = "nvim"
	}
	path, err := exec.LookPath(bin)
	if err != nil {
		return "", "", fmt.Errorf("supported Neovim is required to save: %w", err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	tmp, err := os.MkdirTemp("", "shoshinvim-version-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(tmp)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	boundProcess(cmd)
	cmd.Dir = tmp
	cmd.Env = isolatedEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("Neovim version: %w", err)
	}
	re := regexp.MustCompile(`NVIM v(\d+)\.(\d+)\.(\d+)`)
	m := re.FindStringSubmatch(string(out))
	if len(m) < 4 {
		return "", "", fmt.Errorf("unrecognized Neovim version")
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	if major == 0 && minor < 12 {
		return "", "", fmt.Errorf("Neovim 0.12+ required; found %s", m[0])
	}
	return path, m[0], nil
}

func Probe(ctx context.Context, bin string, files map[string]string) error {
	tmp, err := os.MkdirTemp("", "shoshinvim-probe-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	config := filepath.Join(tmp, "config", "nvim")
	if err = os.MkdirAll(config, 0700); err != nil {
		return err
	}
	var script strings.Builder
	script.WriteString("local ok, err = xpcall(function()\nvim.g.shoshinvim_probe = true\nvim.opt.modeline = false\nvim.opt.loadplugins = false\n")
	fmt.Fprintf(&script, "vim.opt.rtp:prepend(%s)\n", LuaString(config))
	for _, name := range sorted(files) {
		if name == "lua/user_config/user.lua" {
			continue
		}
		p, err := safePath(config, name)
		if err != nil {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return err
		}
		if err = os.WriteFile(p, []byte(files[name]), 0600); err != nil {
			return err
		}
		fmt.Fprintf(&script, "assert(loadfile(%s))\n", LuaString(p))
		if strings.HasPrefix(name, "lua/") && strings.HasSuffix(name, ".lua") {
			module := strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(name, "lua/"), ".lua"), "/", ".")
			fmt.Fprintf(&script, "package.preload[%s] = assert(loadfile(%s))\n", LuaString(module), LuaString(p))
		}
	}
	fmt.Fprintf(&script, "dofile(%s)\n", LuaString(filepath.Join(config, "init.lua")))
	script.WriteString("end, debug.traceback)\nif not ok then io.stderr:write(tostring(err)); vim.cmd('cquit 1') end\nvim.cmd('qa!')\n")
	p := filepath.Join(tmp, "probe.lua")
	if err = os.WriteFile(p, []byte(script.String()), 0600); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--clean", "--headless", "-i", "NONE", "-n", "-l", p)
	boundProcess(cmd)
	cmd.Dir = tmp
	cmd.Env = isolatedEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("isolated Neovim validation failed: %w: %s", err, out)
	}
	return nil
}

// Dependencies reports the programs a state needs; Mason-installed tools are found in target's data directory.
func Dependencies(s State, target string) []Dependency {
	list := []Dependency{{ID: "nvim", Purpose: "Validate generated configuration"}}
	if len(Plugins(s)) > 0 {
		list = append(list, Dependency{ID: "git", Purpose: "Fetch approved plugin revisions"})
	}
	list = append(list, languageDependencies(s)...)
	if linting(s) || debugging(s) {
		list = append(list, runtimes["node"])
		if Value(s, "tools.provider", "system") == "mason" {
			list = append(list, Dependency{ID: "npm", Purpose: "Mason installs JavaScript-based tools with npm"})
		}
	}
	for _, name := range linters(s) {
		list = append(list, Dependency{ID: name, Purpose: map[string]string{"markdownlint": "Markdown linting", "eslint_d": "JavaScript and TypeScript linting"}[name],
			Instruction: "Install " + name + " with npm, or select tools.provider=mason"})
	}
	if debugging(s) {
		list = append(list, Dependency{ID: "js-debug-adapter", Purpose: "JavaScript and TypeScript debugging", Instruction: "Install js-debug-adapter (vscode-js-debug), or select tools.provider=mason"})
	}
	if Value(s, "navigation.picker", "native") != "native" || Value(s, "experience.layout", "native") == "rich" {
		list = append(list, Dependency{ID: "rg", Purpose: "Project text search", Instruction: "Install ripgrep with your package manager; no runtime installation is automatic."})
	}
	if Value(s, "navigation.picker", "native") == "fzf-lua" {
		list = append(list, Dependency{ID: "fzf", Purpose: "fzf-lua fuzzy matching", Instruction: "Install fzf with your package manager (for example: brew install fzf)."})
	}
	if treesitter(s) {
		list = append(list,
			Dependency{ID: "tree-sitter", Purpose: "Compile Tree-sitter parsers", Instruction: "Install tree-sitter-cli 0.26.1 or later with your package manager (for example: brew install tree-sitter-cli), not from npm; or select tools.provider=mason"},
			Dependency{ID: "cc", Purpose: "Compile Tree-sitter parsers", Instruction: "Install a C compiler: Xcode Command Line Tools on macOS, build-essential or gcc on Linux, Visual Studio Build Tools or LLVM on Windows."},
			Dependency{ID: "curl", Purpose: "Download Tree-sitter parser sources"},
			Dependency{ID: "tar", Purpose: "Unpack Tree-sitter parser sources"})
	}
	list = dedupeDependencies(list)
	for i := range list {
		if list[i].ID == "cc" {
			// Any C compiler the tree-sitter CLI can find will do.
			list[i].Status = "missing"
			for _, name := range []string{"cc", "gcc", "clang", "cl", "zig"} {
				if p, err := exec.LookPath(name); err == nil {
					list[i].Path, list[i].Status = p, "present"
					break
				}
			}
			continue
		}
		p, err := exec.LookPath(toolCommand(s, list[i].ID))
		if err == nil && !runtimeWorks(list[i].ID, p) {
			list[i].Status = "missing" // For example macOS's java placeholder, which only explains how to install Java.
		} else if err == nil {
			list[i].Path = p
			list[i].Status = "present"
		} else if toolCommand(s, list[i].ID) != list[i].ID {
			list[i].Status = "missing"
		} else {
			candidate := filepath.Join(DataDirFor(target), "mason", "bin", list[i].ID)
			if runtime.GOOS == "windows" {
				for _, ext := range []string{".cmd", ".exe", ".bat"} {
					if _, err := os.Stat(candidate + ext); err == nil {
						candidate += ext
						break
					}
				}
			}
			if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
				list[i].Path = candidate
				list[i].Status = "present"
			} else {
				list[i].Status = "missing"
			}
		}
	}
	return list
}

// WithNeovim makes the nvim row of deps report the Neovim ShoshinVim uses, given the
// path and error Neovim returned for it (--nvim, the neovim preference or PATH).
// Dependencies only looks on PATH, which may hold another Neovim than the one chosen.
func WithNeovim(deps []Dependency, path string, err error) []Dependency {
	for i := range deps {
		if deps[i].ID != "nvim" {
			continue
		}
		if err == nil {
			deps[i].Path, deps[i].Status, deps[i].Instruction = path, "present", ""
		} else {
			deps[i].Path, deps[i].Status, deps[i].Instruction = "", "missing", "Install Neovim 0.12 or later, or choose one with --nvim"
		}
	}
	return deps
}

// runtimeWorks tells a real runtime from a placeholder on PATH; macOS ships a java
// command that only explains how to install Java.
func runtimeWorks(id, path string) bool {
	if id != "java" {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "-version")
	boundProcess(cmd)
	return cmd.Run() == nil
}
