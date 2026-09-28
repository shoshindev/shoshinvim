package engine

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed data/plugins.json
var pluginData []byte

type Plugin struct {
	Repository string `json:"repository"`
	Revision   string `json:"revision"`
	Reviewed   string `json:"reviewed"`
}

func Actions(s State) ([]Action, error) {
	var pins map[string]Plugin
	if err := json.Unmarshal(pluginData, &pins); err != nil {
		return nil, err
	}
	var out []Action
	ids := Plugins(s)
	if Value(s, "plugins.manager", "none") == "lazy" {
		ids = append([]string{"lazy"}, ids...)
	}
	base, err := Canonical(DataDir())
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		p, ok := pins[id]
		if !ok {
			return nil, fmt.Errorf("missing adapter %s", id)
		}
		out = append(out, Action{ID: id, Kind: "git", Source: "https://github.com/" + p.Repository + ".git", Revision: p.Revision, Destination: filepath.Join(base, "plugins", id), Description: "Git clone " + p.Repository + " at " + p.Revision})
	}
	if Value(s, "tools.provider", "system") == "mason" {
		pkgs := map[string]bool{}
		for _, l := range languages {
			if Enabled(s, "language."+l.ID+".lsp") && toolCommand(s, l.Executable) == l.Executable {
				pkgs[l.Package] = true
			}
		}
		if Value(s, "formatting.provider", "native") == "conform" {
			if toolCommand(s, "prettier") == "prettier" && hasWeb(s) {
				pkgs["prettier"] = true
			}
			if toolCommand(s, "stylua") == "stylua" && Enabled(s, "language.lua.lsp") {
				pkgs["stylua"] = true
			}
		}
		for _, pkg := range sorted(pkgs) {
			out = append(out, Action{ID: "mason:" + pkg, Kind: "mason", Package: pkg, Destination: filepath.Join(base, "mason"), Description: "Mason registry/download/build for " + pkg + " (registry-selected version; may invoke npm or download binaries)"})
		}
	}
	return out, nil
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
func acquire(ctx context.Context, a Action) error {
	base, err := Canonical(DataDir())
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(base, a.Destination)
	if err != nil {
		return err
	}
	dest, err := safePath(base, filepath.ToSlash(rel))
	if err != nil {
		return err
	}
	if a.Kind == "git" {
		if _, err = os.Stat(dest); err == nil {
			cmd := exec.CommandContext(ctx, "git", "-C", dest, "rev-parse", "HEAD")
			boundProcess(cmd)
			b, e := cmd.Output()
			if e == nil && strings.TrimSpace(string(b)) == a.Revision {
				return nil
			}
			return fmt.Errorf("existing plugin differs from reviewed pin; preserve it and review a separate migration: %s", dest)
		} else if !os.IsNotExist(err) {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			return err
		}
		staging, err := os.MkdirTemp(filepath.Dir(dest), ".download-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(staging)
		if err = run(ctx, "", "git", "init", "--quiet", staging); err != nil {
			return err
		}
		if err = run(ctx, staging, "git", "remote", "add", "origin", a.Source); err != nil {
			return err
		}
		if err = run(ctx, staging, "git", "fetch", "--depth=1", "origin", a.Revision); err != nil {
			return err
		}
		if err = run(ctx, staging, "git", "-c", "advice.detachedHead=false", "checkout", "--detach", a.Revision); err != nil {
			return err
		}
		return os.Rename(staging, dest)
	}
	if a.Kind != "mason" {
		return fmt.Errorf("unsupported external action")
	}
	bin, _, err := Neovim(ctx, "")
	if err != nil {
		return err
	}
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
`, LuaString(masonPath), LuaString(dest), LuaString(a.Package), LuaString(a.Package))
	p := filepath.Join(tmp, "install.lua")
	if err = os.WriteFile(p, []byte(script), 0600); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, bin, "--clean", "--headless", "-i", "NONE", "-n", "-l", p)
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
func RunActions(ctx context.Context, target string, actions []Action, progress func(string)) []ActionResult {
	results := make([]ActionResult, len(actions))
	for i, a := range actions {
		results[i] = ActionResult{Action: a, Status: "pending"}
	}
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
			child, cancel := context.WithTimeout(ctx, 5*time.Minute)
			err := acquire(child, a)
			cancel()
			if err != nil {
				r.Status = "failed"
				r.Detail = err.Error()
			} else {
				r.Status = "installed"
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
func Retry(ctx context.Context, target string, progress func(string)) ([]ActionResult, error) {
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
	authorized, err := Actions(*ins.State)
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
	unlock, err := acquireLock(ins.Target)
	if err != nil {
		return nil, err
	}
	defer unlock()
	return RunActions(ctx, ins.Target, failed, progress), nil
}

func hasWeb(s State) bool {
	for _, l := range languages {
		if l.ID != "lua" && Enabled(s, "language."+l.ID+".lsp") {
			return true
		}
	}
	return false
}
