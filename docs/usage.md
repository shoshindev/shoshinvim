# Using ShoshinVim

How the setup works, where it writes, and the command-line interface. The [README](../README.md) covers what ShoshinVim is and how to build it.

## Targets

The `NVIM_APPNAME` must be the same when configuring and launching Neovim. Existing unmanaged configurations are read only; ShoshinVim will not replace them. `--config-dir /absolute/path` selects an explicit target instead: a folder such as `~/.config/work` is started with `NVIM_APPNAME=work nvim` (in PowerShell: `$env:NVIM_APPNAME='work'; nvim`), and a folder outside the configuration home with `nvim -u <folder>/init.lua`. Plugins and tools are stored in the data directory that launch uses, and the final screen shows the exact command. All work in this repository has been tested against temporary targets, not the developer's personal Neovim configuration.

## The setup

Move with the arrow keys (or `j`/`k`), choose with Enter, and go back with Esc; number keys pick an option directly. The Editor section is a single form that holds every text and editing setting — change a row with ←/→, and `r` restores its recommendation. The review summarizes each section, validates the draft in an isolated Neovim, and lists the files and downloads before offering **Save and download** or **Save without downloading**. Downloads never happen merely by reviewing a configuration or starting Neovim. Press `/` anywhere for the command menu: preview the generated Lua (a read-only copy opens in Neovim, one tab per file; `:qa` returns) or the file diff, check dependencies, set any setting, retry failed downloads, or open recovery. Ctrl+C cancels work; pressing it twice quits.

The Workspace section's **Your shortcuts** step lists every key you will have, grouped and labeled with where it comes from (kickstart.nvim, LazyVim, ShoshinVim or yours). Enter changes a key, `d` turns one off, `a` adds one (any action, or any Ex command such as `Lazy`), and `R` restores the style. Conflicts, such as a key that makes a longer one wait, are shown under the list.

## In Neovim

With the recommended kickstart.nvim shortcuts, press **Space** and pause inside Neovim to see what comes next. **Space ?** opens a guide to the keys actually configured; **Space sf** finds files; **Space sg** searches text; **Space Space** lists open buffers; **Space f** formats; **`\`** opens the explorer. The LazyVim style uses LazyVim's keys instead (**Space Space** finds files, **Space e** explorer, **Space cf** formats, **Shift H/L** switches buffers). Completion uses **Ctrl Space** to request, **Ctrl n/p** to select, and **Ctrl y** to accept. Enter keeps its normal behavior. Icons need a Nerd Font: the Appearance step shows a few glyphs, and if they look like boxes or question marks, plain symbols keep every screen readable. Ghostty, WezTerm and kitty include the glyphs, so ShoshinVim recommends icons there or when a Nerd Font is installed.

## CLI

```sh
./bin/shoshinvim doctor
./bin/shoshinvim inspect --output json
./bin/shoshinvim plan --profile rich-ide --out review.json   # scripted preset; a profile replaces earlier values
./bin/shoshinvim plan --state desired-state.json --include-external --out install-review.json
./bin/shoshinvim apply --plan review.json
./bin/shoshinvim apply --plan install-review.json --yes
./bin/shoshinvim retry
./bin/shoshinvim rollback --transaction <id>
./bin/shoshinvim completion zsh
./bin/shoshinvim update
./bin/shoshinvim uninstall
```

Plans bind the target identity, catalog, Neovim version, exact file contents/hashes and external actions. `apply` rebuilds the plan with the Neovim you choose (`--nvim`, the `neovim` preference, or PATH); a plan reviewed with a different Neovim is rejected without running the program it names, because a plan's digest is a checksum, not a signature. `--yes` is accepted only with an exported plan. A stale plan is rejected. `--include-external` authorizes the action list in that plan; inspect it before applying. Exporting refuses to overwrite an existing file.

`--no-alt-screen` asks the same questions line by line with normal terminal scrollback; Enter accepts the recommended answer. `--no-color` / `NO_COLOR` and `--no-progress` support simpler output. `--output json` is available for inspection, health and planning/application results. Commands return nonzero on failed operations; a successful local save can still have failed external actions, reported separately.

## Your files

Generated Lua lives in `init.lua` and the needed modules under `lua/user_config/`. Add manual Lua to **`lua/user_config/user.lua`**, which is created once and never overwritten. Generated files have ownership hashes; changing one blocks saving until you review recovery. `.shoshinvim/` contains declarative state, the ownership manifest, acquisition receipts, and durable transaction journals with backups.

A save locks the target, rechecks the plan, backs up changes, writes adjacent temporary files, verifies generated Lua in isolated Neovim, and commits state. Multi-file saves are recoverable transactions, not an atomic directory swap. **Recovery & history** in the `/` menu lists interrupted or previous transactions. Rollback restores local configuration only; it never deletes acquired tools or plugins. Restoring a generated snapshot archives manual drift first.

Preflight executes controlled generated core code. It does not execute `user.lua`, plugin code or language servers. Isolated XDG directories are not a sandbox for arbitrary Lua. The real plugin/LSP integration test is separate.

Plugin revisions are pinned in the embedded catalog. Explicitly approved Mason downloads use the registry's resolved package versions and record their own receipts. Dependencies live under `shoshinvim/` in the data directory of the Neovim that loads the target, and Mason runs inside your chosen Neovim; manager lockfiles are separate from generated-file ownership. Native startup never bootstraps a plugin manager or installs missing plugins.

Application preferences are optional TOML in `$XDG_CONFIG_HOME/shoshinvim/config.toml` (normally `~/.config/shoshinvim/config.toml` on macOS/Linux):

```toml
neovim = "/absolute/path/to/nvim"
no_color = false
no_alt_screen = false
```

Saved state is JSON with `schema_version: 1` and `catalog_version: "2026.09.28"`. Native values use canonical `options.<name>` IDs. **Set any setting** in the `/` menu accepts `options.<name> <value>` for any Neovim option. Custom executable paths use `tools.path.lua-language-server /absolute/path/to/lua-language-server` (likewise other supported servers, Prettier and StyLua); they take precedence over PATH and reviewed Mason acquisition. Advanced `filetypes` records can be supplied through `plan --state`, as can `keymaps` records (`{"mode": "n", "key": "<leader>ff", "action": "find.files"}`; `"action": "command:Lazy"` runs a command and `"none"` turns a style key off). Incompatible formats require an explicit migration; no silent conversion occurs.
