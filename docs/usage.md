# Using ShoshinVim

This guide covers the setup, the files it writes, changing your configuration later and the command line. The [README](../README.md) covers installing.

## First run

Run `shoshinvim`. It walks through eight sections: Editor, Appearance, Workspace, Navigation, Languages, Formatting, Coding and Git. Every question marks one answer as recommended, and the cursor starts on it, so Enter accepts it.

| Key | What it does |
|---|---|
| ↑ ↓ or `j` `k` | Move |
| Enter | Choose |
| A number | Pick that option directly |
| Esc | Go back |
| `/` | Open the command menu |
| Ctrl+C | Cancel the current work; press it twice to quit |

The Editor section is one form with every text and editing setting. ←/→ changes a row, and `r` puts back its recommended value.

The Workspace section's **Your shortcuts** step lists every key you will have, grouped and labeled with where it comes from: kickstart.nvim, LazyVim, ShoshinVim or you. Enter changes a key, `d` turns it off, `a` adds one (any action, or an Ex command such as `Lazy`), and `R` restores the original style. The list shows conflicts below it, such as a key that makes a longer key wait.

The last step is the review. It summarizes each section, checks the configuration in a separate Neovim that doesn't load your own configuration, and lists every file it would write and everything it would download. It also lists programs your choices need that aren't installed. Then choose:

- **Save and download** writes the files and downloads the plugins, language tools and syntax parsers.
- **Save without downloading** writes only the files. Neovim starts without the missing plugins until you run `shoshinvim` and save again.

Nothing is written or downloaded before you choose one of these. Opening Neovim never downloads anything either. When saving finishes, the last screen shows the command that starts Neovim with your configuration.

The `/` menu works on every screen. From there you can:

- **Preview generated Lua**: open a read-only copy of the files in Neovim, one tab per file. `:qa` returns to the setup.
- **See file changes**: show what a save would change, file by file.
- **Check dependencies**: list the programs your choices need and whether they're installed.
- **Set any setting**: set a value that has no question, such as `options.colorcolumn 80`.
- **Retry failed downloads** and **Recovery & history**, described below.

If the full-screen setup doesn't work in your terminal, `shoshinvim --no-alt-screen` asks the same questions as numbered lists in ordinary terminal output. Enter accepts the recommended answer.

## Where it writes

ShoshinVim writes your configuration to `~/.config/nvim`, the folder Neovim loads by default.

If that folder already holds a configuration ShoshinVim didn't create, such as LazyVim, NvChad or your own, ShoshinVim only reads it. It never changes, imports or merges it. Build a separate configuration next to it with `NVIM_APPNAME`, the environment variable that tells Neovim which folder under `~/.config` to use:

```sh
NVIM_APPNAME=shoshin shoshinvim   # writes ~/.config/shoshin
NVIM_APPNAME=shoshin nvim         # starts Neovim with it
```

Use the same name every time you run either command. Without it, Neovim loads `~/.config/nvim` instead.

`--config-dir /absolute/path` picks any folder. A folder inside `~/.config`, such as `~/.config/work`, starts with `NVIM_APPNAME=work nvim`. A folder anywhere else starts with `nvim -u <folder>/init.lua`.

Plugins and language tools go in Neovim's data folder, under `shoshinvim/`: `~/.local/share/nvim/shoshinvim/` by default, or `~/.local/share/<name>/shoshinvim/` with `NVIM_APPNAME=<name>`.

## In Neovim

With the recommended kickstart.nvim shortcuts, press Space and pause to see which keys come next.

| Keys | Action |
|---|---|
| Space `?` | A guide to the keys you configured |
| Space `sf` | Find files |
| Space `sg` | Search text in the project |
| Space Space | List open buffers |
| Space `f` | Format the file |
| `\` | Open the file explorer |

The LazyVim style uses LazyVim's keys instead: Space Space finds files, Space `e` opens the explorer, Space `cf` formats, and Shift+H and Shift+L switch buffers.

Completion opens with Ctrl+Space. Ctrl+N and Ctrl+P select an item, and Ctrl+Y accepts it. Enter keeps its normal behavior.

Icons need a Nerd Font, a font with extra symbols for file types and Git. The Appearance section shows a few of them. If they look like boxes or question marks, choose plain symbols. Ghostty, WezTerm and kitty include these symbols, so ShoshinVim recommends icons in those terminals or when a Nerd Font is installed.

## Your files

The configuration is ordinary Lua that runs without ShoshinVim:

| File | Who writes it |
|---|---|
| `init.lua` | ShoshinVim. It loads the modules below, then `user.lua`. |
| `lua/user_config/*.lua` | ShoshinVim, one module per area, such as `keymaps.lua` and `appearance.lua`. |
| `lua/user_config/user.lua` | You. ShoshinVim creates it once and never changes it again. |
| `.shoshinvim/` | ShoshinVim: your saved choices, a fingerprint of each generated file, download records and the history of saves with their backups. |

Put your own Lua in `user.lua`. It loads last, so a setting there overrides the generated one:

```lua
vim.o.relativenumber = true
vim.keymap.set('n', '<leader>w', '<cmd>write<cr>', { desc = 'Save file' })
```

Don't edit the generated files. ShoshinVim compares each one with its fingerprint, and if one changed, it stops saving and opens with "needs recovery". To fix that, copy your change into `user.lua`, then open recovery and choose **Restore the files this save wrote**. ShoshinVim first moves the edited files into `.shoshinvim/recovery`, so your edit isn't lost.

## Change your configuration later

Run `shoshinvim` again, with the same `NVIM_APPNAME` or `--config-dir` if you used one. It says "Welcome back" and offers:

- **Review & change** (recommended) opens the review with your saved choices. **Change a section** jumps to one section and brings you back to the review afterwards.
- **Walk through every section** starts from the first question, with your saved answers selected.

Saving again shows the files it changes (**See file changes** in the `/` menu) and backs up each one first.

To undo a save, open **Recovery & history** from the `/` menu. It lists every save with its time and the number of changed files. Choose one, then **Restore the files from before this save**. This restores configuration files only. Downloaded plugins and tools stay on disk.

## Plugins and updates

Each plugin is pinned to an exact commit in ShoshinVim's catalog, and ShoshinVim downloads that commit. lazy.nvim loads the plugins when they're needed, but ShoshinVim sets it up not to install or update anything, so `:Lazy update` leaves them as they are.

Mason installs language tools, such as language servers and formatters, at the versions its registry lists when you approve the download.

`shoshinvim update` updates only ShoshinVim itself. It doesn't change your configuration, plugins or tools. ShoshinVim can't move plugins to newer commits yet.

## Troubleshooting

**The review lists missing programs.** Install them yourself, then run `shoshinvim` and save again. `shoshinvim doctor` prints the same list. Plugins need Git. Web language tools, linters and the JavaScript debugger need Node.js and npm. Project text search uses ripgrep. Syntax parsers need tree-sitter-cli 0.26.1 or later (Mason can download it), a C compiler, curl and tar.

**Some downloads failed.** Choose **Retry failed downloads** on the last screen or in the `/` menu, or run `shoshinvim retry`. Retrying doesn't rewrite your files.

**Neovim starts without plugins.** You saved without downloading, or the downloads failed. Run `shoshinvim` and save again, or retry.

**Icons look like boxes or question marks.** Install a Nerd Font and select it in your terminal, or choose plain symbols in the Appearance section.

**"This folder already holds a Neovim configuration."** ShoshinVim didn't create that configuration, so it won't change it. Use `NVIM_APPNAME` as shown in [Where it writes](#where-it-writes).

**"An earlier save was interrupted, or generated files were edited by hand."** Choose **Open recovery**. If you edited a generated file, follow the steps in [Your files](#your-files). If a save was interrupted, restore the files from before that save, then save again.

**A ShoshinVim error appears when Neovim starts.** Each module loads on its own, so one broken plugin doesn't stop the rest. The message names the module or plugin that failed.

## Command line

| Command | What it does |
|---|---|
| `shoshinvim` | Open the setup |
| `shoshinvim doctor` | Check that the programs your configuration needs are installed |
| `shoshinvim inspect` | Show the target folder and whether ShoshinVim created what's in it |
| `shoshinvim retry` | Retry downloads that failed |
| `shoshinvim rollback` | List earlier saves; `--transaction <id>` restores one after you confirm |
| `shoshinvim plan`, `shoshinvim apply` | Set up a configuration from a script (see below) |
| `shoshinvim update` | Install a newer ShoshinVim after showing its release notes |
| `shoshinvim uninstall` | Remove the `shoshinvim` executable |
| `shoshinvim completion zsh` | Print shell completions (also `bash`, `fish`, `powershell`) |

`update` and `uninstall` act only on a copy that the installer put in place. They never touch your Neovim configuration, plugins or tools.

Every command takes these options:

| Option | What it does |
|---|---|
| `--config-dir /absolute/path` | Use this configuration folder instead of `~/.config/nvim` |
| `--nvim /absolute/path/to/nvim` | Use this Neovim instead of the first one on your PATH |
| `--no-alt-screen` | Ask questions in ordinary terminal output |
| `--no-color` | Turn off colors (so does the `NO_COLOR` environment variable) |
| `--no-progress` | Hide progress output |
| `--output json` | Print results as JSON, for scripts (`inspect`, `doctor`, `plan`, `apply`, `retry`, `rollback`) |

Commands exit with a nonzero status when they fail. A save can succeed while some downloads fail, and ShoshinVim reports those separately.

### Scripted setup

`plan` builds a configuration without the interactive setup and exports it to a file you can read before applying it:

```sh
shoshinvim plan --profile rich-ide --out review.json
shoshinvim apply --plan review.json

shoshinvim plan --state desired-state.json --include-external --out install-review.json
shoshinvim apply --plan install-review.json --yes
```

- `--profile` starts from a preset: `rich-ide`, `minimal`, `web`, `lua` or `custom`. It replaces values set before it.
- `--state` reads your choices from a JSON file (the format is below).
- Without `--include-external`, the plan only writes files. With it, `apply` also runs the downloads listed in the plan, so read that list first.
- `--yes` skips the confirmation. It works only with a plan file.
- `--out` refuses to overwrite an existing file.

A plan records the target folder, the catalog version, the Neovim version, the exact contents of every file and every download. `apply` rebuilds the plan with your Neovim and refuses it if anything differs or the target changed since. The plan's checksum catches accidental changes but doesn't prove who made the file, so `apply` never runs the Neovim a plan names. It uses `--nvim`, the `neovim` preference or the one on your PATH.

### Saved choices

`.shoshinvim/` stores your choices as JSON with `schema_version: 1` and `catalog_version: "2026.09.28"`. ShoshinVim refuses a file from another schema or catalog version instead of converting it.

- Neovim options use `options.<name>`, and **Set any setting** in the `/` menu accepts `options.<name> <value>` for any of them.
- `tools.path.lua-language-server /absolute/path/to/lua-language-server` uses a program you installed instead of a Mason download. The same works for the other supported language servers, Prettier and StyLua. A path set this way wins over PATH and Mason.
- A `--state` file can also hold `filetypes` records and `keymaps` records, such as `{"mode": "n", "key": "<leader>ff", "action": "find.files"}`. `"action": "command:Lazy"` runs a command, and `"action": "none"` turns a style's key off.

### Preferences

Optional preferences go in `~/.config/shoshinvim/config.toml` (or `$XDG_CONFIG_HOME/shoshinvim/config.toml`):

```toml
neovim = "/absolute/path/to/nvim"
no_color = false
no_alt_screen = false
```

## How saving protects your files

A save locks the folder so two saves can't run at once, checks the plan again and backs up every file it will change. Then it writes the new files next to the old ones, checks the new Lua in a separate Neovim and records the save. If a save stops partway, the next run opens recovery. A save that changes several files doesn't happen all at once, but you can always roll it back.

The check before saving runs only the generated Lua. It doesn't run `user.lua`, plugins or language servers. The separate Neovim uses its own temporary folders, but that isn't a security sandbox for Lua you add.
