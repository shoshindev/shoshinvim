# nvim-configurator configuration catalog

- Status: authoritative product scope
- Reference: [`nvim-lua/kickstart.nvim`](https://github.com/nvim-lua/kickstart.nvim)
- Audited upstream commit: [`80743df53d8f7058fc5b60e41f1081d11df9c880`](https://github.com/nvim-lua/kickstart.nvim/tree/80743df53d8f7058fc5b60e41f1081d11df9c880) (2026-09-14)
- Target: Neovim 0.12+

This catalog defines everything `nvim-configurator` must be able to represent, explain, preview, generate, validate, and recover. The catalog covers the complete current Kickstart configuration, its optional example modules, and the native editor choices already defined by this project.

The catalog is comprehensive; onboarding is not. A user may accept a profile and reach Review without visiting every section. Later settings expose every decision independently.

## 1. Scope rules

1. **Kickstart parity is a coverage target, not a blind copy.** Every meaningful Kickstart choice must be representable, but generated code must be validated against the supported Neovim version and the selected plugin versions.
2. **Kickstart defaults and optional examples remain distinct.** Optional modules never become enabled merely because they exist in the catalog.
3. **Every implication is visible.** Selecting a feature shows its plugins, external tools, keymaps, generated files, installation phase, and failure impact.
4. **Local configuration and external acquisition remain separate.** A verified Lua write may succeed even when a plugin, parser, language server, formatter, linter, or debugger installation fails.
5. **No silent external-tool installation.** Mason and parser automation may be selected, but Review must disclose every requested download and require explicit confirmation.
6. **Generated Lua remains ordinary and independent.** Removing the configurator must not break the generated Neovim configuration.
7. **Unmanaged configurations remain read-only.** Complete catalog coverage does not authorize importing or rewriting arbitrary existing Lua.
8. **Upstream drift is intentional, not automatic.** A future Kickstart change updates this catalog only after review; it never silently changes an existing user's selected or persisted state.

Catalog labels:

- `[KICKSTART DEFAULT]` — enabled in the audited Kickstart `init.lua`.
- `[KICKSTART OPTIONAL]` — shipped as a commented example or optional module.
- `[CONDITIONAL]` — enabled only when a dependency or capability is present.
- `[NATIVE]` — Neovim behavior requiring no plugin.
- `[PROJECT BASELINE]` — an existing nvim-configurator choice that remains in scope even when Kickstart does not set it explicitly.

## 2. Profiles

Profiles are starting proposals, not permanent modes.

### Kickstart

Reproduces the audited Kickstart defaults, including its native options, default plugins, mappings, Lua tooling, completion, search, formatting infrastructure, and Treesitter policy. Optional Kickstart modules remain off until explicitly selected.

### General editing

Native editor behavior, diagnostics, a restrained appearance, text objects, surroundings, search, and Git signs. Language-specific tools are not implied.

### Web development

Kickstart-style editor and search behavior plus JavaScript/TypeScript, HTML, CSS, JSON, completion, snippets, formatting choices, and relevant parsers.

### Lua development

Kickstart-style Lua setup with `lua_ls`, Stylua, LuaSnip, completion, Neovim runtime-library awareness, and Lua-related parsers.

### Custom

Begins with native editor defaults and no optional plugins. The user selects sections individually.

Changing a profile previews affected values. Manually changed values are preserved unless the user explicitly chooses to replace them with the new profile.

## 3. Editor

### Native behavior

| Setting | Kickstart proposal | Supported choices |
| --- | --- | --- |
| Lua module cache | On | On; Off |
| Leader key | Space | Any valid non-conflicting mapping |
| Local leader key | Space | Any valid non-conflicting mapping |
| Line number style | Absolute | Absolute; Relative with absolute current line; Off |
| Mouse | All modes | All modes; Normal and Visual; Off |
| Show current mode | Off | On; Off |
| System clipboard | On after UI startup | Use system clipboard; Neovim only |
| Wrap long lines | Neovim default | On; Off |
| Break indent | On | On; Off |
| Highlight current line | On | On; Off |
| Persistent undo | On | On; Off |
| Search case | Smart | Smart; Ignore case; Exact |
| Sign column | Always | Auto; Always; Never |
| Update time | 250 ms | Valid positive duration |
| Mapping timeout | 300 ms | Valid positive duration |
| Vertical split direction | Right | Right; Left |
| Horizontal split direction | Below | Below; Above |
| Visible whitespace | On | On; Off |
| Whitespace characters | `» `, `·`, `␣` | Editable tab, trail, and non-breaking-space text |
| Live substitution preview | Split | Split; Inline; Off |
| Scroll offset | 10 | Non-negative integer |
| Confirm before abandoning changes | On | On; Off |
| Indentation source | Detect automatically | Detect automatically; Explicit |
| Indentation width | Profile-dependent | 2; 4; 8; custom validated value |
| Indent with | Spaces | Spaces; Tabs |

System clipboard activation is delayed until the UI is available when that avoids startup cost. Clipboard-provider availability is tracked separately from the selected intent.

### Editing helpers

- `guess-indent.nvim` `[KICKSTART DEFAULT]` — automatically detect indentation.
- `mini.ai` `[KICKSTART DEFAULT]` — enhanced around/inside text objects, configurable mappings, and search range.
- `mini.surround` `[KICKSTART DEFAULT]` — add, delete, and replace surroundings.
- `nvim-autopairs` `[KICKSTART OPTIONAL]` — automatically insert matching brackets and quotes.
- `indent-blankline.nvim` `[KICKSTART OPTIONAL]` — indentation guides, including blank lines.

## 4. Appearance

- Nerd Font availability flag and plain-text fallback.
- `tokyonight.nvim` `[KICKSTART DEFAULT]`.
- Tokyo Night variant: night, storm, moon, or day.
- Comment italics on or off.
- Built-in colorscheme alternative and extensible validated colorscheme selection.
- `mini.statusline` `[KICKSTART DEFAULT]`.
- Statusline icons on or off.
- Cursor location format.
- `mini.icons` `[CONDITIONAL]` when a Nerd Font is selected.
- `todo-comments.nvim` `[KICKSTART DEFAULT]` with comment signs on or off.

The configurator UI itself continues to use semantic highlight links rather than forcing its own colorscheme.

## 5. Diagnostics

| Decision | Kickstart proposal |
| --- | --- |
| Update while inserting | Off |
| Sort by severity | On |
| Float border | Rounded |
| Show source | When several sources exist |
| Underline threshold | Warning and above |
| Virtual text | On |
| Virtual lines | Off |
| Open details after diagnostic jump | On |
| Diagnostic list action | Location list |

The catalog also supports disabling diagnostics, choosing quickfix instead of location list, configuring float focus, and selecting severity thresholds independently.

## 6. Keymaps and automatic behavior

Every keymap record contains action ID, modes, key sequence, scope, description, source feature, enabled state, and conflict status. Generated feature-local mappings are buffer-local whenever possible.

### Native defaults

| Key | Action |
| --- | --- |
| `Esc` | Clear search highlighting |
| `<leader>q` | Open diagnostic location list |
| `Esc Esc` | Leave terminal mode |
| `Ctrl-h/j/k/l` | Move between windows |

Optional native mappings include moving windows with `Ctrl-Shift-h/j/k/l` and disabling arrow keys in Normal mode.

### Automatic behavior

- Highlight yanked text.
- Run feature-specific actions on LSP attach/detach.
- Build selected plugins after `vim.pack` install/update.
- Attach and optionally install Treesitter parsers on filetype detection.
- Run selected linters on configured events.
- Open and close debugging UI with debugger lifecycle events.

## 7. Plugin management

`vim.pack` is the generated plugin mechanism. The adapter supports:

- Git source URL.
- Friendly display name.
- Default branch, named branch, tag/version range, or exact commit.
- Required and optional dependencies.
- Conditional inclusion.
- Post-install/update build command.
- Lockfile entry and availability state.
- Requested, downloading, installed, update available, failed, and unknown states.
- Retry without rewriting verified local configuration.
- Removal planning without silently deleting plugin data.

The Review screen lists Git use, resolved plugin directories, lockfile changes, build commands, and network actions.

## 8. Search and navigation

### Telescope `[KICKSTART DEFAULT]`

Plugins:

- `nvim-lua/plenary.nvim`
- `nvim-telescope/telescope.nvim`
- `nvim-telescope/telescope-ui-select.nvim`
- `nvim-telescope/telescope-fzf-native.nvim` `[CONDITIONAL]` when `make` is available

Actions:

- Search help.
- Search keymaps.
- Find files.
- Select a Telescope picker.
- Search the current word or visual selection.
- Live grep.
- Search diagnostics.
- Resume the last picker.
- Search recent files.
- Search commands.
- Search open buffers.
- Fuzzy-search the current buffer.
- Grep only open files.
- Search Neovim configuration files.
- Find LSP references, implementations, definitions, document symbols, workspace symbols, and type definitions.

Configuration includes picker mappings, dropdown theme, preview visibility, transparency, native sorter availability, and UI-select integration.

### File explorer

- Built-in file commands.
- `neo-tree.nvim` `[KICKSTART OPTIONAL]` with `plenary.nvim` and `nui.nvim`.
- Reveal-current-file mapping and close mapping.
- Filesystem window behavior.

Search/picker and file-tree choices are independent.

## 9. Git

`gitsigns.nvim` and its full recommended action set are `[KICKSTART DEFAULT]` in the current upstream snapshot.

Configurable signs:

- Add.
- Change.
- Delete.
- Top delete.
- Change-delete.

Configurable actions:

- Next and previous hunk with native-diff fallback.
- Stage and reset hunk in Normal or Visual mode.
- Stage and reset buffer.
- Preview hunk normally or inline.
- Blame current line.
- Diff against the index or previous commit.
- Send repository or current-file hunks to quickfix.
- Toggle current-line blame.
- Toggle word-level diff.
- Select a hunk as a text object.

## 10. Which-key

`which-key.nvim` is `[KICKSTART DEFAULT]`.

- Enable or disable.
- Popup delay.
- Mapping icons with Nerd Font fallback.
- Search, Toggle, Git Hunk, and LSP Action groups.
- Groups derived from generated mappings rather than maintained as an unrelated second source of truth.

## 11. Languages and LSP

Infrastructure:

- Neovim native LSP client.
- `nvim-lspconfig`.
- `mason.nvim`.
- `mason-lspconfig.nvim`.
- `mason-tool-installer.nvim`.
- `fidget.nvim` for progress.

Each language record supports server choice, external command, filetypes, root markers, server settings, initialization behavior, capabilities, installation provider, dependency status, formatting ownership, and parser/formatter/linter/debugger companions.

Kickstart coverage includes:

- Lua: `lua_ls` and Stylua `[KICKSTART DEFAULT]`.
- C/C++: `clangd` example.
- Go: `gopls` example.
- Python: `pyright` example.
- TypeScript: current upstream example name `tsc`.
- Rust: `rust_analyzer` example, with Rustaceanvim mentioned as an alternative.

Lua behavior includes LuaJIT runtime paths, Neovim runtime-library awareness, suppression of third-party workspace prompts, respect for `.luarc.json`/`.luarc.jsonc`, and disabling `lua_ls` formatting when Stylua owns formatting.

Mason may install selected servers and tools only after the Review screen lists them and the user confirms the external phase.

### LSP actions

- Rename.
- Code action in Normal and Visual mode.
- Go to declaration.
- References.
- Implementation.
- Definition.
- Type definition.
- Document symbols.
- Workspace symbols.
- Document-reference highlighting and cleanup.
- Inlay-hint toggle when supported.

## 12. Completion and snippets

`blink.cmp` and LuaSnip are `[KICKSTART DEFAULT]`. `friendly-snippets` is `[KICKSTART OPTIONAL]`.

Completion choices:

- Native completion.
- `blink.cmp`.
- None.

Blink settings:

- Default, Super-Tab, Enter, or no preset mappings.
- LSP, path, and snippet sources.
- LuaSnip integration.
- Documentation auto-show and delay.
- Nerd Font Mono or normal spacing.
- Lua fuzzy matching or the optional Rust matcher.
- Signature help.

The UI spells out the resulting completion keys and warns before mapping Enter to acceptance.

## 13. Formatting

`conform.nvim` is `[KICKSTART DEFAULT]`.

- Manual asynchronous formatting in Normal or Visual mode.
- Per-filetype formatter lists.
- Sequential formatters.
- First-available formatter fallback.
- Optional format on save by filetype.
- Save timeout.
- LSP fallback or LSP formatting disabled.
- Error-notification policy.

Kickstart examples include Stylua, `rustfmt`, `isort`, `black`, `prettierd`, and `prettier`. Example lines do not become selected defaults merely because they appear upstream.

## 14. Linting

`nvim-lint` is `[KICKSTART OPTIONAL]`.

- Per-filetype linter selection.
- Multiple linters.
- Run on buffer entry, after write, and/or after leaving Insert mode.
- Skip non-modifiable buffers.
- Manual lint action.

The shipped example enables Markdown with `markdownlint` and documents possible linters for Clojure, Dockerfile, Inko, Janet, JSON, Markdown, reStructuredText, Ruby, Terraform, and plain text.

## 15. Treesitter

`nvim-treesitter` on its `main` branch is `[KICKSTART DEFAULT]`.

- Parser selection and installation state.
- Baseline parsers: Bash, C, Diff, HTML, Lua, LuaDoc, Markdown, Markdown inline, Query, Vim, and VimDoc.
- Automatic installation for recognized filetypes.
- Highlighting.
- Indentation when an indent query exists.
- Optional expression-based folding.
- Safe buffer-validity check after asynchronous installation.
- Parser update build hook.

The catalog may expose additional parsers returned by the supported `nvim-treesitter` version, but generated state stores explicit parser IDs rather than depending on a remote catalog at startup.

## 16. Debugging

The Go-focused DAP example is `[KICKSTART OPTIONAL]`.

Plugins:

- `nvim-dap`
- `nvim-dap-ui`
- `nvim-nio`
- `mason-nvim-dap.nvim`
- `nvim-dap-go`
- Existing `mason.nvim`

Behavior:

- Install and configure Delve.
- Start or continue.
- Step into, over, and out.
- Toggle breakpoint.
- Conditional breakpoint.
- Toggle debugging UI.
- Open/close UI with debugger lifecycle.
- Plain and Nerd Font breakpoint/control icons.
- Windows-specific attached Delve mode.

Debuggers are selected per language. Go is the first complete recipe, not a permanent product limit.

## 17. Dependency and health catalog

Base checks:

- Neovim 0.12+.
- Git.
- `make`.
- `unzip`.
- C compiler.
- `rg`.
- `fd`.
- Tree-sitter CLI.
- System clipboard provider.
- Optional Nerd Font.

Feature checks:

- Selected language runtimes and package managers such as Node/npm, Go, Rust/Cargo, or Python.
- Selected language servers.
- Selected formatters.
- Selected linters.
- Selected debugger adapters.
- Plugin build tools.

Health reports purpose, status, impact, provider, last-check time, and the exact selected feature affected. A missing optional dependency never becomes a generic configuration failure.

## 18. Plugin inventory

### Default or conditional Kickstart repositories

1. `NMAC427/guess-indent.nvim`
2. `lewis6991/gitsigns.nvim`
3. `folke/which-key.nvim`
4. `folke/tokyonight.nvim`
5. `folke/todo-comments.nvim`
6. `nvim-mini/mini.nvim`
7. `nvim-lua/plenary.nvim`
8. `nvim-telescope/telescope.nvim`
9. `nvim-telescope/telescope-ui-select.nvim`
10. `nvim-telescope/telescope-fzf-native.nvim` `[CONDITIONAL]`
11. `j-hui/fidget.nvim`
12. `neovim/nvim-lspconfig`
13. `mason-org/mason.nvim`
14. `mason-org/mason-lspconfig.nvim`
15. `WhoIsSethDaniel/mason-tool-installer.nvim`
16. `stevearc/conform.nvim`
17. `L3MON4D3/LuaSnip`
18. `saghen/blink.cmp`
19. `nvim-treesitter/nvim-treesitter`

### Optional Kickstart repositories

20. `rafamadriz/friendly-snippets`
21. `windwp/nvim-autopairs`
22. `mfussenegger/nvim-dap`
23. `rcarriga/nvim-dap-ui`
24. `nvim-neotest/nvim-nio`
25. `jay-babu/mason-nvim-dap.nvim`
26. `leoluz/nvim-dap-go`
27. `lukas-reineke/indent-blankline.nvim`
28. `mfussenegger/nvim-lint`
29. `nvim-neo-tree/neo-tree.nvim`
30. `MunifTanjim/nui.nvim`

Custom plugins are allowed only through the user-owned extension file in the initial product. A generic remote marketplace, arbitrary setup-schema inference, and automatic two-way editing of custom plugin code are outside this catalog.

## 19. Generated file groups

The deterministic generator may emit:

- `init.lua`
- `lua/user_config/editor.lua`
- `lua/user_config/diagnostics.lua`
- `lua/user_config/keymaps.lua`
- `lua/user_config/appearance.lua`
- `lua/user_config/plugins.lua`
- `lua/user_config/search.lua`
- `lua/user_config/git.lua`
- `lua/user_config/languages.lua`
- `lua/user_config/completion.lua`
- `lua/user_config/formatting.lua`
- `lua/user_config/linting.lua`
- `lua/user_config/treesitter.lua`
- `lua/user_config/debugging.lua`
- `lua/user_config/user.lua`, created once and never overwritten
- `nvim-pack-lock.json` when plugin state is recorded

Only files required by the selected state are created. Every generated module is safe to require when its optional plugin or external tool is unavailable; the affected feature becomes pending rather than making Neovim fail to start.

## 20. Settings navigation

After onboarding, the settings workspace contains:

1. Overview
2. Profiles
3. Editor
4. Appearance
5. Diagnostics
6. Keymaps
7. Search
8. File explorer
9. Git
10. Languages and LSP
11. Completion and snippets
12. Formatting
13. Linting
14. Treesitter
15. Debugging
16. Dependencies and Health
17. Plugins
18. Recovery

Review, Generated Lua, Diff, key help, and removal guidance remain contextual utilities rather than catalog sections.

## 21. Explicit boundaries

The complete Kickstart catalog does not include:

- AI assistants or cloud synchronization.
- A generic plugin marketplace.
- Automatic import or merge of an unmanaged configuration.
- Automatic adoption of manually changed generated files.
- Silent shell/package-manager commands outside an explicitly reviewed external phase.
- Credentials written into generated configuration.
- Live application of draft settings.
- Deleting unused plugin data as part of local configuration rollback.

These boundaries preserve a transparent, reversible configuration generator while allowing full Kickstart-level capability coverage.
