package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Actions lists the downloads a state needs, placed in the data directory Neovim uses for target.
func Actions(s State, target string) ([]Action, error) {
	var out []Action
	ids := Plugins(s)
	if Value(s, "plugins.manager", "none") == "lazy" {
		ids = append([]string{"lazy"}, ids...)
	}
	base, err := Canonical(DataDirFor(target))
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		repo, rev, dir, err := source(s, id)
		if err != nil {
			return nil, err
		}
		out = append(out, Action{ID: id, Kind: "git", Source: "https://github.com/" + repo + ".git", Revision: rev, Destination: filepath.Join(base, "plugins", dir), Description: "Git clone " + repo + " at " + rev})
	}
	if Value(s, "tools.provider", "system") == "mason" {
		pkgs := languagePackages(s)
		if treesitter(s) {
			pkgs["tree-sitter-cli"] = true
		}
		for _, name := range linters(s) {
			pkgs[name] = true // Mason's package names match the programs: markdownlint, eslint_d.
		}
		if debugging(s) {
			pkgs["js-debug-adapter"] = true
		}
		for _, pkg := range sorted(pkgs) {
			out = append(out, Action{ID: "mason:" + pkg, Kind: "mason", Package: pkg, Destination: filepath.Join(base, "mason"), Description: "Mason registry/download/build for " + pkg + " (registry-selected version; may invoke npm or download binaries)"})
		}
	}
	// Parsers come last: they need nvim-treesitter and, with Mason, its tree-sitter CLI.
	if parsers := parserNames(Parsers(s)); len(parsers) > 0 {
		out = append(out, Action{ID: "treesitter:parsers", Kind: "parsers", Parsers: parsers, Destination: filepath.Join(base, "parsers"),
			Description: "Tree-sitter parsers " + strings.Join(parsers, ", ") + " (source archives at the revisions nvim-treesitter pins, compiled with tree-sitter-cli and your C compiler)"})
	}
	return out, nil
}

// Installed reports whether an action's result is already on disk: a cloned plugin or
// a Mason package. Running an installed action again is harmless but not needed.
func Installed(a Action) bool {
	var marker string
	switch a.Kind {
	case "git":
		marker = filepath.Join(a.Destination, ".git")
	case "mason":
		marker = filepath.Join(a.Destination, "packages", a.Package)
	case "parsers":
		for _, lang := range a.Parsers {
			if _, err := os.Stat(filepath.Join(a.Destination, "parser", lang+".so")); err != nil {
				return false
			}
		}
		return len(a.Parsers) > 0
	default:
		return false
	}
	_, err := os.Stat(marker)
	return err == nil
}

func run(ctx context.Context, dir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	boundProcess(cmd)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) > 8000 {
			out = out[len(out)-8000:]
		}
		return fmt.Errorf("%s: %w: %s", name, err, out)
	}
	return nil
}

// head returns the commit checked out in a downloaded plugin, or "" when there is none.
func head(ctx context.Context, dir string) string {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return ""
	}
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD")
	boundProcess(cmd)
	b, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// acquire performs one approved action and returns a note on anything besides the
// download itself. nvim is the Neovim the user chose; Mason runs inside it, so a
// different or older Neovim on PATH is never used.
func acquire(ctx context.Context, base, nvim string, a Action) (string, error) {
	rel, err := filepath.Rel(base, a.Destination)
	if err != nil {
		return "", err
	}
	dest, err := safePath(base, filepath.ToSlash(rel))
	if err != nil {
		return "", err
	}
	if a.Kind == "git" {
		if _, err = os.Stat(dest); err == nil {
			if head(ctx, dest) == a.Revision {
				return "", nil
			}
			return "", fmt.Errorf("existing plugin differs from reviewed pin; preserve it and review a separate migration: %s", dest)
		} else if !os.IsNotExist(err) {
			return "", err
		}
		if a.ID == "lazy" {
			// Earlier versions downloaded lazy.nvim to a folder where it does not count itself
			// installed. That checkout moves when it is the pin, leaving a link so files those
			// versions generated still find it; lazy.nvim ignores links in its folder. Any
			// other checkout there is the user's and stays.
			legacy := filepath.Join(filepath.Dir(dest), legacyLazyFolder)
			if head(ctx, legacy) == a.Revision {
				if err = os.Rename(legacy, dest); err != nil {
					return "", err
				}
				if os.Symlink(lazyFolder, legacy) == nil {
					return "moved from plugins/" + legacyLazyFolder + ", which now links to it", nil
				}
				// No links here (Windows without Developer Mode): download beside it instead.
				if err = os.Rename(dest, legacy); err != nil {
					return "", err
				}
			}
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			return "", err
		}
		staging, err := os.MkdirTemp(filepath.Dir(dest), ".download-")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(staging)
		if err = run(ctx, "", "git", "init", "--quiet", staging); err != nil {
			return "", err
		}
		if err = run(ctx, staging, "git", "remote", "add", "origin", a.Source); err != nil {
			return "", err
		}
		if err = run(ctx, staging, "git", "fetch", "--depth=1", "origin", a.Revision); err != nil {
			return "", err
		}
		if err = run(ctx, staging, "git", "-c", "advice.detachedHead=false", "checkout", "--detach", a.Revision); err != nil {
			return "", err
		}
		return "", os.Rename(staging, dest)
	}
	if a.Kind == "parsers" {
		return "", compileParsers(ctx, base, nvim, dest, a.Parsers)
	}
	if a.Kind != "mason" {
		return "", fmt.Errorf("unsupported external action")
	}
	if nvim == "" {
		return "", fmt.Errorf("Mason needs a verified Neovim")
	}
	return "", installMason(ctx, base, dest, nvim, a.Package)
}

// installMason installs one Mason package with the pinned mason.nvim inside nvim.
func installMason(ctx context.Context, base, dest, nvim, pkg string) error {
	tmp, err := os.MkdirTemp("", "shoshinvim-mason-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	masonPath := filepath.Join(base, "plugins", "mason")
	script := fmt.Sprintf(`vim.fn.mkdir(vim.fn.stdpath('log'), 'p')
vim.opt.rtp:prepend(%s)
require('mason').setup({ install_root_dir = %s })
local registry = require('mason-registry')
local finished, failure = false, nil
local function install(ok)
  local worked, err = xpcall(function()
  if not ok then failure = 'Registry refresh failed'; finished = true; return end
  local valid, pkg = pcall(registry.get_package, %s)
  if not valid then failure = tostring(pkg); finished = true; return end
  if pkg:is_installed() then finished = true; return end
  local handle = pkg:install()
  handle:once('closed', function() if not pkg:is_installed() then failure = 'Package install failed; inspect Mason logs' end; finished = true end)
  end, debug.traceback)
  if not worked then failure = tostring(err); finished = true end
end
if registry.has_package(%s) then install(true) else registry.refresh(install) end
if not vim.wait(240000, function() return finished end, 100) then failure = 'Package installation timed out' end
if failure then io.stderr:write(failure); vim.cmd('cquit 1') end
vim.cmd('qa!')
`, LuaString(masonPath), LuaString(dest), LuaString(pkg), LuaString(pkg))
	p := filepath.Join(tmp, "install.lua")
	if err = os.WriteFile(p, []byte(script), 0600); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, nvim, "--clean", "--headless", "-i", "NONE", "-n", "-l", p)
	boundProcess(cmd)
	cmd.Dir = tmp
	cmd.Env = isolatedEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if log, e := os.ReadFile(filepath.Join(tmp, "state", "nvim", "mason.log")); e == nil {
			if len(log) > 6000 {
				log = log[len(log)-6000:]
			}
			out = append(out, log...)
		}
		return fmt.Errorf("Mason: %w: %s", err, out)
	}
	return nil
}

// compileParsers installs Tree-sitter parsers with the pinned nvim-treesitter inside the
// user's chosen Neovim. A tree-sitter CLI that Mason downloaded comes first on PATH.
func compileParsers(ctx context.Context, base, nvim, dest string, parsers []string) error {
	if nvim == "" {
		return fmt.Errorf("parsers need a verified Neovim")
	}
	plugin := filepath.Join(base, "plugins", "nvim-treesitter")
	if _, err := os.Stat(filepath.Join(plugin, "lua", "nvim-treesitter", "install.lua")); err != nil {
		return fmt.Errorf("nvim-treesitter is not downloaded; retry once it is")
	}
	tmp, err := os.MkdirTemp("", "shoshinvim-parsers-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	var langs []string
	for _, p := range parsers {
		langs = append(langs, LuaString(p))
	}
	script := fmt.Sprintf(`vim.fn.mkdir(vim.fn.stdpath('cache'), 'p')
vim.opt.rtp:prepend(%s)
if vim.fn.executable('tree-sitter') ~= 1 then
  io.stderr:write('tree-sitter CLI not found: install tree-sitter-cli 0.26.1 or later (not from npm), or choose Mason for language tools')
  vim.cmd('cquit 1')
end
local ts = require('nvim-treesitter')
ts.setup({ install_dir = %s })
local langs = { %s }
local ok, err = pcall(function() return ts.install(langs, { max_jobs = 4 }):wait(840000) end)
local missing = {}
for _, lang in ipairs(langs) do
  if not vim.uv.fs_stat(%s .. '/parser/' .. lang .. '.so') then missing[#missing + 1] = lang end
end
if not ok or #missing > 0 then
  io.stderr:write((ok and '' or tostring(err) .. '\n') .. 'Parsers not installed: ' .. table.concat(missing, ', '))
  vim.cmd('cquit 1')
end
vim.cmd('qa!')
`, LuaString(plugin), LuaString(dest), strings.Join(langs, ", "), LuaString(dest))
	p := filepath.Join(tmp, "parsers.lua")
	if err = os.WriteFile(p, []byte(script), 0600); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, nvim, "--clean", "--headless", "-i", "NONE", "-n", "-l", p)
	boundProcess(cmd)
	cmd.Dir = tmp
	cmd.Env = isolatedEnv(tmp)
	for i, kv := range cmd.Env {
		if key, value, _ := strings.Cut(kv, "="); strings.EqualFold(key, "PATH") {
			cmd.Env[i] = key + "=" + filepath.Join(base, "mason", "bin") + string(os.PathListSeparator) + value
		}
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		if len(out) > 8000 {
			out = out[len(out)-8000:]
		}
		return fmt.Errorf("Tree-sitter: %w: %s", err, out)
	}
	return nil
}

func RunActions(ctx context.Context, target, nvim string, actions []Action, progress func(string)) []ActionResult {
	results := make([]ActionResult, len(actions))
	for i, a := range actions {
		results[i] = ActionResult{Action: a, Status: "pending"}
	}
	base, baseErr := Canonical(DataDirFor(target))
	p, pathErr := safePath(target, ".shoshinvim/acquisition.json")
	persist := func() error {
		if pathErr != nil {
			return pathErr
		}
		return AtomicWrite(p, JSON(results), 0600)
	}
	if err := persist(); err != nil {
		return append(results, ActionResult{Status: "failed", Detail: "Could not persist acquisition intent: " + err.Error()})
	}
	for i, a := range actions {
		r := ActionResult{Action: a, Status: "pending"}
		if ctx.Err() != nil {
			r.Detail = "Cancelled after local commit"
		} else {
			if progress != nil {
				progress(a.Description)
			}
			limit := 5 * time.Minute
			if a.Kind == "parsers" {
				limit = 15 * time.Minute // Large grammars such as TypeScript take a while to compile.
			}
			child, cancel := context.WithTimeout(ctx, limit)
			note, err := "", baseErr
			if err == nil {
				note, err = acquire(child, base, nvim, a)
			}
			cancel()
			if err != nil {
				r.Status = "failed"
				r.Detail = err.Error()
			} else {
				r.Status, r.Detail = "installed", note
			}
		}
		results[i] = r
		if err := persist(); err != nil {
			return append(results, ActionResult{Status: "failed", Detail: "Could not persist acquisition progress: " + err.Error()})
		}
		if progress != nil {
			progress("[" + strings.ToUpper(r.Status) + "] " + a.ID + " " + r.Detail)
		}
	}
	return results
}

// Retry reruns approved actions that did not finish. bin selects Neovim as --nvim does.
func Retry(ctx context.Context, target, bin string, progress func(string)) ([]ActionResult, error) {
	ins, err := Inspect(target)
	if err != nil {
		return nil, err
	}
	if ins.Ownership != "SAVED" || ins.State == nil {
		return nil, fmt.Errorf("retry requires an intact saved configuration")
	}
	p, err := safePath(ins.Target, ".shoshinvim/acquisition.json")
	if err != nil {
		return nil, err
	}
	var prior []ActionResult
	if err = DecodeFile(p, &prior); os.IsNotExist(err) {
		return nil, fmt.Errorf("no downloads are pending; save with downloads to install plugins and tools")
	} else if err != nil {
		return nil, err
	}
	authorized, err := Actions(*ins.State, ins.Target)
	if err != nil {
		return nil, err
	}
	var failed []Action
	for _, r := range prior {
		if r.Status == "installed" {
			continue
		}
		found := false
		for _, a := range authorized {
			if string(JSON(a)) == string(JSON(r.Action)) {
				failed = append(failed, a)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("external plan changed; review new actions")
		}
	}
	nvim := ""
	for _, a := range failed {
		if a.Kind == "mason" {
			if nvim, _, err = Neovim(ctx, bin); err != nil {
				return nil, err
			}
			break
		}
	}
	unlock, err := acquireLock(ins.Target)
	if err != nil {
		return nil, err
	}
	defer unlock()
	return RunActions(ctx, ins.Target, nvim, failed, progress), nil
}
