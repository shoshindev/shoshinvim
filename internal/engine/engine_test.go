package engine

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fixture(t *testing.T) (string, State) {
	t.Helper()
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("requires Neovim 0.12+")
	}
	s, err := LoadCatalog().Profile(NewState(), "minimal")
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(t.TempDir(), "nvim"), s
}
func planFor(t *testing.T, target string, s State) Plan {
	t.Helper()
	p, err := BuildPlan(context.Background(), target, s, "", false)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func applyFor(t *testing.T, p Plan) ApplyResult {
	t.Helper()
	r, err := Apply(context.Background(), p, ApplyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !r.LocalCommitted {
		t.Fatal("not committed")
	}
	return r
}
func TestProfilesPreserveManualValues(t *testing.T) {
	c := LoadCatalog()
	s := NewState()
	if err := c.Set(&s, "editor.wrap", true, true); err != nil {
		t.Fatal(err)
	}
	s, err := c.Profile(s, "web")
	if err != nil {
		t.Fatal(err)
	}
	if s.Values["options.wrap"] != true {
		t.Fatal("profile overwrote explicit choice")
	}
	if err = c.Validate(s); err != nil {
		t.Fatal(err)
	}
}
func TestProfileReplacesPreviousProfile(t *testing.T) {
	c := LoadCatalog()
	rich, _ := c.Profile(NewState(), "rich-ide")
	minimal, err := c.Profile(rich, "minimal")
	if err != nil {
		t.Fatal(err)
	}
	if plugins := Plugins(minimal); len(plugins) > 0 || Value(minimal, "plugins.manager", "none") != "none" {
		t.Fatalf("minimal kept rich-ide plugins: %v", plugins)
	}
	if err = c.Validate(minimal); err != nil {
		t.Fatal(err)
	}
	if err = c.Set(&minimal, "navigation.picker", "snacks", true); err != nil {
		t.Fatal(err)
	}
	web, _ := c.Profile(minimal, "web")
	lua, _ := c.Profile(web, "lua")
	if lua.Values["navigation.picker"] != "snacks" || Enabled(lua, "language.typescript.lsp") {
		t.Fatalf("explicit choice lost or web languages leaked: %v", lua.Values)
	}
	if err = c.Validate(lua); err != nil {
		t.Fatal(err)
	}
}
func TestProfilesGenerateValidDeterministicLua(t *testing.T) {
	fixture(t)
	for name := range LoadCatalog().Profiles {
		t.Run(name, func(t *testing.T) {
			s, _ := LoadCatalog().Profile(NewState(), name)
			a, err := Generate(s)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := Generate(Clone(s))
			if string(JSON(a)) != string(JSON(b)) {
				t.Fatal("nondeterministic")
			}
			bin, _, err := Neovim(context.Background(), "")
			if err != nil {
				t.Fatal(err)
			}
			if err = Probe(context.Background(), bin, a); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestLuaEscaping(t *testing.T) {
	target, s := fixture(t)
	s.Values["options.titlestring"] = "quote\" slash\\ newline\n Հայերեն '); os.execute('touch BAD') --"
	planFor(t, target, s)
}
func TestApplyReopenDriftAndPreserveUser(t *testing.T) {
	target, s := fixture(t)
	applyFor(t, planFor(t, target, s))
	ins, err := Inspect(target)
	if err != nil || ins.Ownership != "SAVED" {
		t.Fatalf("%+v %v", ins, err)
	}
	user := filepath.Join(target, filepath.FromSlash(userPath))
	os.WriteFile(user, []byte("-- manual user extension\n"), 0600)
	s.Values["options.wrap"] = true
	applyFor(t, planFor(t, target, s))
	b, _ := os.ReadFile(user)
	if string(b) != "-- manual user extension\n" {
		t.Fatal("overwrote user file")
	}
	os.WriteFile(filepath.Join(target, "init.lua"), []byte("-- manual work\n"), 0600)
	if _, err = BuildPlan(context.Background(), target, s, "", false); err == nil {
		t.Fatal("accepted drift")
	}
}
func TestStaleAndTamperedPlans(t *testing.T) {
	target, s := fixture(t)
	p := planFor(t, target, s)
	applyFor(t, p)
	if _, err := Apply(context.Background(), p, ApplyOptions{}); err == nil {
		t.Fatal("accepted stale plan")
	}
	p = planFor(t, target, s)
	p.Files[0].Content = "evil"
	p.Digest = p.Fingerprint()
	if _, err := Apply(context.Background(), p, ApplyOptions{}); err == nil {
		t.Fatal("accepted tampered plan")
	}
}
func TestUnmanagedAndSymlinks(t *testing.T) {
	target, s := fixture(t)
	os.MkdirAll(target, 0700)
	os.WriteFile(filepath.Join(target, "init.lua"), []byte("-- mine"), 0600)
	if _, err := BuildPlan(context.Background(), target, s, "", false); err == nil {
		t.Fatal("accepted unmanaged config")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skip(err)
	}
	if _, err := BuildPlan(context.Background(), link, s, "", false); err == nil {
		t.Fatal("accepted symlink")
	}
}
func TestRollbackWriteFailures(t *testing.T) {
	for _, phase := range []string{"backed-up", "wrote:.shoshinvim/manifest.json", "wrote:.shoshinvim/state.json", "wrote:lua/user_config/options.lua", "verified"} {
		t.Run(phase, func(t *testing.T) {
			target, s := fixture(t)
			applyFor(t, planFor(t, target, s))
			old, _ := os.ReadFile(filepath.Join(target, "lua/user_config/options.lua"))
			s.Values["options.wrap"] = true
			p := planFor(t, target, s)
			_, err := Apply(context.Background(), p, ApplyOptions{Hook: func(at string) error {
				if at == phase {
					return errors.New("injected disk failure")
				}
				return nil
			}})
			if err == nil {
				t.Fatal("fault not injected")
			}
			now, _ := os.ReadFile(filepath.Join(target, "lua/user_config/options.lua"))
			if string(now) != string(old) {
				t.Fatal("not restored")
			}
			ins, e := Inspect(target)
			if e != nil || ins.Ownership != "SAVED" {
				t.Fatalf("%+v %v", ins, e)
			}
		})
	}
}
func TestInitialFailureCanRetry(t *testing.T) {
	target, s := fixture(t)
	_, err := Apply(context.Background(), planFor(t, target, s), ApplyOptions{Hook: func(at string) error {
		if at == "wrote:init.lua" {
			return errors.New("failure")
		}
		return nil
	}})
	if err == nil {
		t.Fatal("no fault")
	}
	applyFor(t, planFor(t, target, s))
}
func TestIncompleteTransactionDetected(t *testing.T) {
	target, s := fixture(t)
	p := planFor(t, target, s)
	j := Journal{ID: "123", Target: p.Target, Status: "prepared", Plan: p}
	if err := saveJournal(&j); err != nil {
		t.Fatal(err)
	}
	ins, err := Inspect(target)
	if err != nil || ins.Ownership != "RECOVERY" {
		t.Fatalf("%+v %v", ins, err)
	}
}
func TestRecoveryArchivesDrift(t *testing.T) {
	target, s := fixture(t)
	r := applyFor(t, planFor(t, target, s))
	manual := "-- irreplaceable manual work\n"
	os.WriteFile(filepath.Join(target, "init.lua"), []byte(manual), 0600)
	if err := RecoverGenerated(target, r.Transaction); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(target, ".shoshinvim/recovery/*/init.lua"))
	if len(matches) != 1 {
		t.Fatal("missing manual archive")
	}
	b, _ := os.ReadFile(matches[0])
	if string(b) != manual {
		t.Fatal("lost content")
	}
}
func TestConcurrentTargetLock(t *testing.T) {
	target, _ := fixture(t)
	unlock, err := acquireLock(target)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if other, e := acquireLock(target); e == nil {
		other()
		t.Fatal("second writer acquired lock")
	}
}
func TestStartupDoesNotBootstrapPlugins(t *testing.T) {
	s, _ := LoadCatalog().Profile(NewState(), "web")
	files, err := Generate(s)
	if err != nil {
		t.Fatal(err)
	}
	p := files["lua/user_config/plugins.lua"]
	if !strings.Contains(p, "missing = false") || strings.Contains(p, "git clone") {
		t.Fatal("unsafe startup")
	}
}
func TestInvalidState(t *testing.T) {
	c := LoadCatalog()
	s := NewState()
	s.Values["navigation.picker"] = "snacks"
	if c.Validate(s) == nil {
		t.Fatal("missing manager")
	}
	s = NewState()
	s.Values["options.channel"] = float64(5)
	if c.Validate(s) == nil {
		t.Fatal("readonly option")
	}
	s = NewState()
	s.Filetypes = map[string]map[string]any{"lua": {"shell": "sh"}}
	if c.Validate(s) == nil {
		t.Fatal("global option in filetype")
	}
	s = NewState()
	s.Keymaps = []Keymap{{"n", "x", "save"}, {"n", "x", "terminal"}}
	if c.Validate(s) == nil {
		t.Fatal("mapping conflict")
	}
}
func FuzzLuaString(f *testing.F) {
	for _, s := range []string{"", "hello", "\"\\\n", "Հայերեն"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := LuaString(s)
		if !strings.HasPrefix(got, "\"") || !strings.HasSuffix(got, "\"") {
			t.Fatal("unquoted")
		}
		if strings.ContainsAny(got, "\n\r\x00") {
			t.Fatal("raw controls")
		}
	})
}

func TestReviewDoesNotWriteConfiguredLogOrExecuteUserLua(t *testing.T) {
	target, s := fixture(t)
	outside := filepath.Join(t.TempDir(), "must-not-exist.log")
	s.Values["options.verbosefile"] = outside
	applyFor(t, planFor(t, target, s))
	os.WriteFile(filepath.Join(target, filepath.FromSlash(userPath)), []byte("error('USER LUA MUST NOT RUN DURING PROBE')"), 0600)
	planFor(t, target, s)
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatal("review created verbose log")
	}
}
func TestExternalCancellationKeepsLocalCommit(t *testing.T) {
	target, _ := fixture(t)
	s, _ := LoadCatalog().Profile(NewState(), "web")
	p := planFor(t, target, s)
	p.RunExternal = true
	p.Digest = p.Fingerprint()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := Apply(ctx, p, ApplyOptions{Progress: func(s string) {
		if s == "LOCAL_COMMITTED" {
			cancel()
		}
	}})
	if err != nil || !r.LocalCommitted {
		t.Fatalf("%+v %v", r, err)
	}
	for _, a := range r.External {
		if a.Status != "pending" {
			t.Fatal("cancelled work misreported")
		}
	}
	ins, err := Inspect(target)
	if err != nil || ins.Ownership != "SAVED" {
		t.Fatalf("%+v %v", ins, err)
	}
}
func TestExplicitConfigStartsWithoutConfigurator(t *testing.T) {
	target, s := fixture(t)
	applyFor(t, planFor(t, target, s))
	tmp := t.TempDir()
	cmd := exec.Command("nvim", "--headless", "-i", "NONE", "-n", "-u", filepath.Join(target, "init.lua"), "+lua assert(vim.o.number); assert(vim.o.wrap == false)", "+qa")
	cmd.Dir = tmp
	cmd.Env = isolatedEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil || strings.Contains(string(out), "Error") {
		t.Fatalf("%s %v", out, err)
	}
}

func TestCustomExecutableAndScopedAcquisition(t *testing.T) {
	c := LoadCatalog()
	s, _ := c.Profile(NewState(), "lua")
	if err := c.Set(&s, "tools.path.lua-language-server", filepath.Join(t.TempDir(), "lua-language-server"), true); err != nil {
		t.Fatal(err)
	}
	if err := c.Set(&s, "tools.path.stylua", "relative/path", true); err == nil {
		t.Fatal("accepted relative executable")
	}
	actions, err := Actions(s, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range actions {
		if a.ID == "mason:lua-language-server" || a.ID == "mason:prettier" {
			t.Fatalf("unneeded acquisition: %s", a.ID)
		}
	}
	files, err := Generate(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files["lua/user_config/lsp.lua"], LuaString(toolCommand(s, "lua-language-server"))) {
		t.Fatal("custom command not emitted")
	}
}

func TestMetadataFollowsGeneratedValidation(t *testing.T) {
	target, s := fixture(t)
	p := planFor(t, target, s)
	_, err := Apply(context.Background(), p, ApplyOptions{Hook: func(phase string) error {
		if strings.HasPrefix(phase, "wrote:lua/") {
			if h, _ := fileHash(target, statePath); h != "" {
				t.Fatal("persisted state changed before generated validation")
			}
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoldenProfiles(t *testing.T) {
	for name := range LoadCatalog().Profiles {
		t.Run(name, func(t *testing.T) {
			s, err := LoadCatalog().Profile(NewState(), name)
			if err != nil {
				t.Fatal(err)
			}
			files, err := Generate(s)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", name+".json")
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err = os.MkdirAll("testdata", 0755); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(path, JSON(files), 0644); err != nil {
					t.Fatal(err)
				}
			}
			expected, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(expected) != string(JSON(files)) {
				t.Fatal("generated output changed; review then refresh the golden fixture")
			}
		})
	}
}

// fakeNeovim is an executable that records each invocation and claims to be Neovim.
func fakeNeovim(t *testing.T) (bin, marker string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell script as a stand-in executable")
	}
	dir := t.TempDir()
	bin, marker = filepath.Join(dir, "nvim"), filepath.Join(dir, "invocations")
	script := "#!/bin/sh\necho \"$@\" >> '" + marker + "'\necho 'NVIM v0.12.5'\n"
	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return bin, marker
}

func TestApplyNeverRunsTheNeovimNamedInAPlan(t *testing.T) {
	target, s := fixture(t)
	fake, marker := fakeNeovim(t)
	p := planFor(t, target, s)
	p.Neovim = fake // An edited plan file; its digest is only a checksum, so it can be refreshed.
	p.Digest = p.Fingerprint()
	if _, err := VerifyPlan(context.Background(), p, ""); err == nil || !strings.Contains(err.Error(), "reviewed with Neovim") {
		t.Fatalf("edited Neovim path accepted: %v", err)
	}
	if _, err := Apply(context.Background(), p, ApplyOptions{}); err == nil {
		t.Fatal("apply accepted a plan made for another Neovim")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("the Neovim named in the plan was executed")
	}
	if ins, _ := Inspect(target); ins.Ownership != "NEW" {
		t.Fatalf("target changed: %s", ins.Ownership)
	}
	// Choosing that Neovim explicitly (--nvim) is the user's decision and is honored.
	if _, err := VerifyPlan(context.Background(), p, fake); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("the explicitly chosen Neovim was not used")
	}
}

func TestMasonUsesTheChosenNeovim(t *testing.T) {
	fake, marker := fakeNeovim(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	target := filepath.Join(t.TempDir(), "nvim")
	base, err := Canonical(DataDirFor(target))
	if err != nil {
		t.Fatal(err)
	}
	a := Action{ID: "mason:stylua", Kind: "mason", Package: "stylua", Destination: filepath.Join(base, "mason"), Description: "Mason test"}
	results := RunActions(context.Background(), target, fake, []Action{a}, nil)
	if len(results) != 1 || results[0].Status != "installed" {
		t.Fatalf("%+v", results)
	}
	b, err := os.ReadFile(marker)
	if err != nil || !strings.Contains(string(b), "--headless") || !strings.Contains(string(b), "-l") {
		t.Fatalf("Mason did not run in the chosen Neovim: %q %v", b, err)
	}
}

func TestDataDirectoryFollowsTheTarget(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("NVIM_APPNAME", "unrelated") // The environment ShoshinVim runs in must not decide.
	if err := os.MkdirAll(filepath.Join(root, "config"), 0700); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(root, "config", "work", "web")
	outside := filepath.Join(root, "projects", "nvim")
	if app := AppName(inside); app != "work/web" {
		t.Fatalf("app name %q", app)
	}
	if app := AppName(outside); app != "" {
		t.Fatalf("outside target got app name %q", app)
	}
	want := func(parts ...string) string {
		p, _ := resolveExisting(filepath.Join(append([]string{root, "data"}, parts...)...))
		return p
	}
	suffix := func(app string) string {
		if runtime.GOOS == "windows" {
			return app + "-data"
		}
		return app
	}
	if got, _ := resolveExisting(DataDirFor(inside)); got != want("work", suffix("web"), "shoshinvim") {
		t.Fatalf("inside data dir %s", got)
	}
	if got, _ := resolveExisting(DataDirFor(outside)); got != want(suffix("nvim"), "shoshinvim") {
		t.Fatalf("outside data dir %s", got)
	}
	s, _ := LoadCatalog().Profile(NewState(), "rich-ide")
	actions, err := Actions(s, inside)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range actions {
		if !strings.HasPrefix(a.Destination, want("work", suffix("web"), "shoshinvim")) {
			t.Fatalf("%s downloads outside the target's data directory: %s", a.ID, a.Destination)
		}
	}
	// Neovim itself must agree: generated Lua reads stdpath('data') .. '/shoshinvim'.
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("requires Neovim 0.12+ for the stdpath check")
	}
	for target, app := range map[string]string{inside: AppName(inside), outside: ""} {
		cmd := exec.Command("nvim", "--headless", "--clean", "--cmd", "lua io.stdout:write(vim.fn.stdpath('data'))", "+qa")
		cmd.Env = append(os.Environ(), "NVIM_APPNAME="+app)
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		got, _ := resolveExisting(filepath.Join(strings.TrimSpace(string(out)), "shoshinvim"))
		if expected, _ := resolveExisting(DataDirFor(target)); got != expected {
			t.Fatalf("Neovim reads plugins from %s, ShoshinVim downloads to %s", got, expected)
		}
	}
}

func TestOneFailingModuleDoesNotDisableTheRest(t *testing.T) {
	_, s := fixture(t)
	files, err := Generate(s)
	if err != nil {
		t.Fatal(err)
	}
	files["lua/user_config/options.lua"] = "error('simulated plugin failure')\n"
	files[userPath] = "-- user extensions\n"
	bin, _, err := Neovim(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if err = Probe(context.Background(), bin, files); err == nil {
		t.Fatal("validation must still reject a failing module")
	}
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(p, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	tmp := t.TempDir()
	cmd := exec.Command(bin, "--headless", "-i", "NONE", "-n", "-u", filepath.Join(dir, "init.lua"),
		"+lua vim.wait(200); io.stdout:write(vim.fn.exists(':ShoshinFormat') .. '|' .. vim.api.nvim_exec2('messages', { output = true }).output)", "+qa!")
	cmd.Dir = tmp
	cmd.Env = isolatedEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "2|") || !strings.Contains(string(out), "options did not load") {
		t.Fatalf("later modules must load and the failure must be reported: %v %q", err, out)
	}
}

// The option snapshot ships in every executable, so it must not carry defaults that
// describe the machine it was taken on: home and temporary folders, install prefixes
// or the snapshot user's shell. Such defaults are recorded as null.
func TestOptionDefaultsAreNotMachineSpecific(t *testing.T) {
	markers := []string{"/Users/", "/home/", "/var/folders/", "/private/var/", "/opt/homebrew", "/usr/local/Cellar", "/nix/store", `C:\`, "/bin/zsh", "/bin/bash"}
	for name, o := range LoadCatalog().Options {
		value, ok := o.Default.(string)
		if !ok {
			continue
		}
		for _, m := range markers {
			if strings.Contains(value, m) {
				t.Errorf("options.json default for %s describes the snapshot machine (%q): record it as null", name, value)
			}
		}
	}
}
