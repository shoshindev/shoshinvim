# ShoshinVim v0.1 interface specification

> Staged implementation: see [current alpha coverage and release gates](implementation-status.md). This document specifies the shipped guided-section interface. The [configuration catalog](configuration-catalog.md) also lists planned capabilities; a listed plugin or installation channel does not imply released support.

- Status: authoritative interaction, safety, state, and responsive-design specification
- Audience: the developer implementing the standalone terminal application
- Product line: **Build a Neovim you understand, one decision at a time.**
- Feature scope: [configuration catalog](configuration-catalog.md)
- Interface direction: keyboard-first terminal UI with the feel of Grok Build — selectable menus, quiet hints, a command palette — without a chat transcript
- Implementation: `internal/ui` (`steps.go` holds every section, question and recommendation; `model.go`, `view.go` and `jobs.go` hold state, rendering and background work; `plain.go` is line mode)

This document specifies behavior and presentation, not production code. ShoshinVim is a deterministic local application. It borrows the look and keyboard grammar of modern coding-agent TUIs without requiring an AI model, cloud service, account, or autonomous tool use.

## 1. Product direction

### Primary interaction model

Running `shoshinvim` opens a fullscreen guided setup:

1. A welcome panel states what will happen and that nothing is written before review.
2. Eight sections — **Editor, Appearance, Workspace, Navigation, Languages, Formatting, Coding, Git** — each ask a short series of decisions.
3. Every decision offers curated options and marks exactly one as **recommended**; the cursor starts there, so Enter accepts it.
4. A review summarizes each section, validates the draft in an isolated Neovim, and lists files and downloads before any write.
5. Saving shows live progress; a final screen says how to start Neovim.

There are **no presets** in the interactive setup. The user walks through every section and decides; recommendations make that fast without deciding for them. Accepting every recommendation yields a cohesive Web and Lua workspace.

ShoshinVim is a standalone executable. It calls Neovim only for isolated validation, health probes and approved Mason installs; the user never needs to open Neovim to configure it. The application owns the alternate screen, restores terminal state on every exit path, and offers `--no-alt-screen` line mode for constrained terminals and captured logs.

### Patterns adopted

- One decision per screen, with numbered options and a `❯` pointer.
- A visible recommendation on every decision, never a hidden default.
- A section sidebar (or progress bar) so the user always knows where they are and what remains.
- One form for many small related settings (the Editor section) instead of a long question chain.
- A review screen as the only path to saving.
- A `/` command palette for power features; `?` for keys.
- Quiet, dim key hints in a footer; a one-line notice for warnings.
- Live progress with a spinner; `Ctrl+C` cancels work, and pressing it twice quits.

### Patterns not adopted

- Profiles or presets in the interactive setup (scripted presets remain a CLI-only affordance).
- A chat transcript, free-text composer, or natural-language interpretation.
- Model selection, token or cost displays, sessions, or agents.
- Silent dependency installation or downloads before review.
- Dashboard cards, three-column layouts, or permanent detail panes.
- Decorative animation beyond the progress spinner.

## 2. Interface anatomy

### 2.1 Layout

Wide terminals (92 columns or more) show a section sidebar:

```text
 ◯ ShoshinVim                                        ~/.config/nvim · new

 ✓ Editor         │  Appearance · 1 of 4
                  │
 ● Appearance     │  Color scheme
                  │  The colors of the whole editor.
 ○ Workspace      │
                  │  ❯ 1. Tokyo Night  ◆ recommended   ●●●●●●
 ○ Navigation     │       Deep night palette with vivid, readable syntax
                  │    2. Habamax                      ●●●●●●
 ○ Languages      │       Muted dark theme · built in
 ...              │    ...

 ● notice line (only when there is something to say)
 ↑↓ move · enter choose · esc back · / commands · ? help
```

Narrower terminals replace the sidebar with a two-line progress header:

```text
 Appearance · 1 of 4                                   step 2 of 8
 ━━━━━━━━━━━━━━━━━━━━━━━━──────────────────────────────────────────
```

The rendered frame always fills the terminal height exactly and no line exceeds its width.

### 2.2 Header

The first line shows the `◯` mark, the gradient product name, and — right-aligned — the shortened target path and a status tag. When space is short the path is dropped before the tag. The welcome screen omits the header because its panel carries the logo.

Status tags:

| Tag | Meaning |
| --- | --- |
| `new` | No managed configuration exists at the target. |
| `saved` | The draft equals the saved configuration. |
| `unsaved changes` | A saved configuration exists and the draft differs. |
| `read-only` | An existing configuration was not created by ShoshinVim. |
| `needs recovery` | A save was interrupted or generated files were edited by hand. |

### 2.3 Sidebar

One row per section plus Review. Marks: `✓` visited, `●` current, `○` not yet visited, `–` skipped. A section is marked skipped only once the decision it depends on has been made (for example, Formatting after choosing no languages). Rows are double-spaced when height allows.

### 2.4 Content

Each screen opens with the section name and position (`Appearance · 1 of 4`), then the decision in bold, a one-line dim explanation, and the options. Option lists adapt to height in three steps: label and detail with spacing; label and detail; labels only with the selected option's detail below. If even that does not fit, the list scrolls to keep the cursor visible.

### 2.5 Notice and footer

The notice line carries transient messages — quit confirmation, errors, "Set options.colorcolumn to 80 in your draft" — and clears on the next key. The footer lists the keys that matter on the current screen in dim text.

### 2.6 Overlays and viewers

Overlays replace the content area and keep the draft and screen underneath:

- **Command palette** (`/`): a panel with a filter input and the command list (Preview generated Lua, See file changes, Plugins used, Check dependencies, Show all settings as JSON, Set any setting, Go to review, Retry failed downloads, Recovery & history, Keys & help, Quit).
- **Set any setting**: an input accepting `setting value`, with up to six ID suggestions and Tab completion.

Full-screen viewers show long content with a title, scroll percentage and `esc` to close: file diff (colored), dependencies, all settings as JSON, and keys & help. **Preview generated Lua** opens Neovim instead: a read-only copy of every file in a temporary folder, one tab per file, started with `nvim --clean -R` so the user's configuration and plugins stay out of it and Neovim's own syntax colors apply. `:qa` returns to the setup and the copy is removed; without a usable Neovim, the Lua is shown in a viewer.

## 3. Setup structure

### 3.1 Question kinds

| Kind | Used for | Interaction |
| --- | --- | --- |
| Single choice | Most decisions | `↑↓` or a number selects; Enter or the number confirms. Long lists are grouped under headings (for example plugin themes and built-in themes). |
| Multiple choice | Languages | `space` or a number toggles `[✓]`; Enter continues. |
| Form row | Editor settings | All rows on one screen under group headings (Display, Wrapping, Indentation, Search, Editing, Windows, Timing); `←→` cycles a row's value; Enter continues. |
| Shortcut list | Your shortcuts (§3.9) | Every shortcut with its source; change, turn off, add or restore. |
| Text input | Theme repository and colorscheme name (§3.8) | Type and paste freely (`/` and `?` are text here); Enter continues; Esc goes back. |

Every option sets explicit catalog values, so the saved state records each decision even when it matches Neovim's default.

### 3.2 Recommendations

- Every single-choice decision and form row has exactly one recommended option; a multiple-choice decision states its recommended set.
- On a new target the cursor (or form value) starts at the recommendation. On a saved target it starts at the saved choice; a value set outside the curated options displays as `custom`.
- Options show `◆ recommended` (or `(recommended)` without color). In the Editor form, a row that differs shows `changed · recommended: <value>`, and `r` restores the recommendation.
- Recommendations never lock alternatives and never overwrite a saved choice.
- **What gets recommended:** the option today's leading starter configurations ship (kickstart.nvim, LazyVim, NvChad, AstroNvim), with GitHub stars breaking ties. Each option's plugin line shows the evidence (stars and which of those projects use it), so the reasoning is checkable.
- **Editor rows** take kickstart.nvim's `init.lua` value where it sets one, otherwise LazyVim's default; the form's help line says so.
- **Every section offers real alternatives**, including a built-in or "none" choice wherever Neovim can do without a plugin.

### 3.3 Sections and recommended choices

| Section | Decisions (recommended first) |
| --- | --- |
| **Editor** (form, 27 rows) | *Display:* Line numbers on · Relative numbers off · Cursor line highlight · Sign column always · Invisible characters tabs & spaces · Column guide off · Empty lines blank. *Wrapping:* Wrap long lines off · Wrapped lines tidy (break at words, keep indentation). *Indentation:* Indent with spaces · Tab width 2 · Indent width 2 · Detect indentation guess-indent.nvim / off. *Search:* Search case smart · Replace preview split. *Editing:* Persistent undo on · Clipboard shared · Mouse all modes · Highlight on copy on · Reopen at last position on · Trim trailing spaces off · Spell check prose on. *Windows:* New splits right & below · Context lines 10 · Context columns 8 · Unsaved changes ask. *Timing:* Key timing fast (300 ms / 250 ms) |
| **Appearance** | Color scheme: Tokyo Night, 25 more theme plugins, 11 themes built into Neovim, or another theme from GitHub (§3.8) · Variant, when the theme has several (light variants marked) · Status line: lualine.nvim / mini.statusline / built in · Open files: bufferline.nvim / barbar.nvim / mini.tabline / no tabs · Indent guides: indent-blankline.nvim / Snacks indent / mini.indentscope / none · Notifications: Snacks notifier / fidget.nvim / built in · Popup borders: rounded / square / none |
| **Workspace** | Shortcut style: kickstart.nvim / LazyVim / Neovim defaults only · Leader key: Space / Comma / Backslash · Your shortcuts (§3.9) · Shortcut hints: which-key.nvim / mini.clue / none · Start screen: Snacks dashboard / mini.starter / none |
| **Navigation** | Find files and text: Telescope / Snacks picker / fzf-lua / mini.pick / built in · File explorer: neo-tree.nvim / nvim-tree.lua / Snacks explorer / oil.nvim / mini.files / netrw |
| **Languages** | Language tools: mason.nvim / I'll install them · Which languages (§3.11): 24 in five groups — *Web* TypeScript & JavaScript, HTML, CSS, JSON, Svelte; *Scripting* Lua & Neovim config, Python, Ruby, PHP, Bash & shell, Elixir; *Systems* Go, Rust, C & C++, Zig, Swift; *JVM & .NET* Java, Kotlin, C#; *Config & docs* YAML, TOML, Markdown, Dockerfile, Terraform — with TypeScript, HTML, CSS, JSON and Lua recommended · Syntax highlighting: nvim-treesitter / built in (§3.10) · Completion: blink.cmp / nvim-cmp / built in / off · Errors and warnings: inline / signs only · Server settings: nvim-lspconfig / no plugin (basic recipes) |
| **Formatting** | Formatter: conform.nvim with each chosen language's formatter (§3.11) / language server · When to format: every save / when I ask |
| **Coding** | Brackets and quotes: nvim-autopairs / mini.pairs / off · Surround: mini.surround / nvim-surround / off · Text objects: mini.ai / built in · TODO comments: todo-comments.nvim / mini.hipatterns / off · Linting: off / nvim-lint (markdownlint; ESLint in projects configured for it) · Debugger (with TypeScript): off / nvim-dap + nvim-dap-ui / nvim-dap + nvim-dap-view |
| **Git** | Git changes: gitsigns.nvim / mini.diff / off |

Plugin options are labeled with the plugin's own name. Each carries a detail line describing what it does and a plugin line (§3.7). Theme options show a color swatch recorded from the real theme.

### 3.4 Skipped decisions

When the user has decided on languages and selected none, completion, diagnostics, server settings and both Formatting decisions are skipped, and their values are removed from the draft. The language-tools decision comes first in Languages and is always asked; Mason is installed only when something needs a separate program (a language, Tree-sitter parsers, a linter or the debugger). The debugger is skipped when TypeScript is not chosen, since its adapter is for JavaScript and TypeScript. The Variant step is skipped for themes with a single variant; the custom-theme steps appear only after choosing "Another theme from GitHub…". Skipped sections are omitted from navigation and summarized as "skipped".

### 3.5 Derived settings

Implementation details follow from decisions and are never asked:

- `plugins.manager` is lazy.nvim exactly when a chosen option needs a plugin, otherwise none.
- lualine.nvim sets `showmode` off and a global status line (`laststatus` 3); mini.statusline sets `showmode` off; the built-in status line keeps Neovim's defaults.
- Built-in completion sets `completeopt` and `pumheight`; blink.cmp and nvim-cmp manage their own menus and tell language servers which completion features they support.
- Choosing a listed theme removes custom-theme settings; with no languages, the completion engine is removed too.
- The leader key is set whenever a shortcut style or leader is chosen; the LazyVim style also sets the local leader to `\`, as LazyVim does.
- Configurations saved before the Workspace choices existed keep their behavior: the earlier "Space shortcuts" layout maps to the LazyVim style with Space as leader, the Snacks start screen, indent guides and notifier.
- Editor behaviors (highlight on copy, trim, reopen at last position, spell check) are autocommands in one `ShoshinEditor` group in `autocmds.lua`.
- Each plugin's setup runs after lazy.nvim has loaded every plugin, and lazy.nvim keeps the configuration folder on the runtime path, so configurations started with `nvim -u` load completely.
- lazy.nvim is downloaded to `plugins/lazy.nvim`, the folder name it looks for, so `:Lazy` lists it as installed and nothing to clean. Configurations that earlier versions downloaded to `plugins/lazy` keep loading from there. The next download moves that folder when it holds the pinned commit and leaves a link in its place, so files an earlier version generated, such as ones restored from history, still find lazy.nvim; lazy.nvim ignores links in its folder. Where links cannot be made, or the folder holds another commit, it stays as it is and lazy.nvim is downloaded anew.
- `termguicolors` is left to Neovim's own terminal detection.

### 3.6 Summaries

The review and "Change a section" list summarize each section on one line using short option names, for example `Tokyo Night · Lualine · Buffer tabs · Rounded borders`. The Editor summary reports `27 settings · all recommended` or the number changed; the Workspace summary includes the number of shortcut changes.

### 3.7 Transparency

Every plugin ShoshinVim can install is described in its registry (`internal/engine/data/plugins.json`): repository, pinned commit, review date, stars and last update on that date, license, which of kickstart.nvim, LazyVim, NvChad and AstroNvim use it (verified in their source), and the plugins it requires.

- Each plugin option shows `owner/repo (+ requirements) · ★ stars · updated Month Year · license · used by …`. A missing license is shown as "no license declared", never hidden.
- The review lists every plugin that will be installed; plugins not in the registry are marked "not reviewed".
- **Plugins used** (review action and `/` command) shows, for each plugin, its link, pinned commit, review status, stars, update date, license, users, and **why** it is included: the decision that chose it, or the plugin that requires it.
- Choosing a plugin never silently adds a feature from another plugin; for example Snacks' own finder and explorer stay off unless chosen, so two plugins never compete for the same job.

### 3.8 A theme from GitHub

The last color-scheme option, "Another theme from GitHub…", accepts any repository:

1. The user types `owner/name` (a full GitHub URL also works).
2. On Enter, ShoshinVim reads metadata only — the current commit (`git ls-remote`) and, through the GitHub API, the repository's stars, last update, license and the colorschemes in its `colors` folder at that commit. Nothing is downloaded or executed. Esc cancels the lookup.
3. The user picks one of the colorschemes found, or types a name when GitHub cannot list them.
4. The theme is pinned to that commit and stored in a folder named after the repository and commit, so a later commit never collides with it.

The review and Plugins used mark it "not reviewed by ShoshinVim". If it fails to load, Neovim keeps a built-in theme and shows a message. Choosing a listed theme later removes the custom theme settings.

### 3.9 Shortcuts

Shortcuts are credited, visible and editable before anything is written.

- **Styles are copied from their source.** kickstart.nvim's keys come from its `init.lua`; LazyVim's from its published keymap list. Each style option names its source link and how many shortcuts it has. "Neovim defaults only" adds none; Neovim's built-in keys (`grn`, `gra`, `grr`, `gri`, `gO`, `K`, `[d`, `]d`) work in every style and are listed under the shortcut list as a reminder.
- **Only what works is listed.** A shortcut whose feature is not chosen is dropped, for example Git-change keys when Git changes is off; finder and explorer shortcuts call whichever finder or explorer was chosen.
- **Your shortcuts** lists every key grouped as Find, Files, Code, Git, Debug, Buffers, Windows, Editor and Yours, with its source (`kickstart.nvim`, `LazyVim`, `ShoshinVim` or `yours`). Keys are spelled in words with the chosen leader, for example `Space sf`, `Ctrl+h`, `Shift+h`; modes other than normal are named after the description (`· normal & visual`).
- Later steps that shortcuts reach name the keys actually configured, for example `Your shortcuts: Space sf find files · Space sg search text in the project` on Find files and text; nothing is named when no style is chosen.
- **Changing** a key asks for Vim notation (`<leader>ff`, `<C-s>`, `]q`). Turning one off records it as off, so a later style update does not bring it back. **Adding** one picks from the same actions, or "Run a command…" for any Ex command such as `Lazy`. `R` restores the style and removes every change.
- **Conflicts** are shown under the list: a key that is also the start of a longer key in the same mode, which Neovim runs only after a pause. LazyVim's `gr` is exempt because LazyVim maps it with `nowait`. A key cannot be used twice: changing or adding a key replaces whatever was on it.
- The generated `keymaps.lua` names the style and its source, then lists shortcuts under `-- From kickstart.nvim`, `-- ShoshinVim additions` and `-- Your shortcuts` headings. ShoshinVim adds only `<leader>?`, the shortcut guide (`:ShoshinHelp`), which lists the keys actually configured. which-key.nvim and mini.clue show the style's group names (for example `Search` for kickstart's `Space s`).

### 3.10 Syntax parsers and coding helpers

- **Tree-sitter parsers are compiled once, during the approved download**, never when Neovim starts (kickstart.nvim installs them on first use of a file type; ShoshinVim does not, because nothing downloads without review). The set is kickstart's base list — bash, c, diff, html, lua, luadoc, markdown, markdown_inline, query, vim, vimdoc — plus javascript, jsdoc, tsx, typescript for TypeScript, css and scss for CSS, and json for JSON.
- Parser sources are the archives at the revisions the pinned nvim-treesitter commit names; Plugins used lists each parser with its repository and revision. Compiling needs tree-sitter-cli 0.26.1 or later (Mason downloads it when Mason installs tools), a C compiler, curl and tar; missing programs are listed in the review.
- The generated `syntax.lua` starts Tree-sitter for every file type that has a parser and uses nvim-treesitter's indentation where it has indent rules.
- mini.surround follows the shortcut style: kickstart's `sa`/`sd`/`sr`, or LazyVim's `gsa`/`gsd`/`gsr`. mini.ai uses kickstart's settings (`aa`/`ii` for the next object, keeping Neovim 0.12's own `an`/`in`).
- Linting runs each linter only where it applies and only when it is installed: markdownlint for Markdown, and eslint_d for JavaScript and TypeScript only inside a project with an ESLint configuration. `:ShoshinLint` lints the current file.
- The debugger uses js-debug-adapter (the VS Code JavaScript debugger) with "Launch this file with Node.js" and "Attach to a running Node.js process". Debugger shortcuts come from the style: kickstart's F5/F1/F2/F3, Space b, Space B and F7; LazyVim's Space d… keys. Both debugger views use icons that need no Nerd Font.

### 3.11 Languages

- Every language is one entry in `internal/engine/data/languages.json`: its server (nvim-lspconfig name, program, Mason package, file types, project markers), its formatter when one is common, its Tree-sitter parsers, and the runtimes its tools need. Generation, Mason downloads, dependency checks, parsers and the interface all read that entry.
- Mason is asked about first, so each language's detail line says what it gets: "pyright + Ruff, installed by Mason; needs Node.js", or without Mason "Needs pyright-langserver and ruff on your PATH". Swift's server comes with Xcode or the Swift toolchain, so the line says so.
- Runtimes (Node.js, Python, Go, Rust, Java (25 or later for Java, 21 or earlier for Kotlin), .NET SDK, Ruby, Elixir, Zig, Terraform) are never installed; the review lists the ones missing for the chosen languages.
- **Server settings.** With nvim-lspconfig (recommended; every leading starter uses it) its maintained recipe is used as is. ShoshinVim adds only its own settings: Neovim's runtime for the Lua server, Swift's server limited to Swift files (its recipe also claims C and C++, which clangd serves), and a custom executable path when one is set. Without the plugin, ShoshinVim writes a basic recipe per server with the same program, file types and project markers, plus the options formatting needs (for example `provideFormatter` for the HTML, CSS and JSON servers).
- **Formatters** (conform.nvim): Prettier for web files, YAML and Markdown; StyLua; Ruff; goimports; rustfmt (from rustup); clang-format; ktlint; shfmt; Taplo. Languages without one are formatted by their server.

## 4. Core flows

### 4.1 First launch

The welcome panel shows the logo, the tagline, the eight section names, the recommendation convention and "Nothing is written or downloaded until you review and save." Below it: the target and status tag, then **Start setup** (recommended) and Quit.

### 4.2 Walking the sections

Enter on a choice records it in the in-memory draft and moves to the next decision; the last decision of a section moves to the next section, and the last section opens the review. Esc goes back one decision — across section boundaries, and from the first decision to the welcome screen. Answers are kept when going back. Leaving the Editor form with Enter or Esc keeps its rows.

### 4.3 Review

Opening the review builds a fresh, non-mutating plan in the background ("Checking your configuration with Neovim…"), which validates generated Lua in an isolated Neovim. The screen shows:

- one summary line per section;
- **Files**: new, updated and removed counts and the target;
- **Downloads**: plugin and language-tool counts;
- **You install**: programs the user must provide (for example ripgrep, fzf for fzf-lua, Node.js, or a language runtime such as Java). Tools that Mason will download are not listed.
- **Plugins**: every plugin that will be installed, by name (§3.7).

Then the actions, in order and as applicable:

| Action | When |
| --- | --- |
| **Save and download** (recommended) | Files change and downloads exist. |
| Save without downloading | Files change and downloads exist; missing downloads stay pending. |
| **Save** (recommended) | Files change and nothing needs downloading. |
| **Download plugins and tools** (recommended) | Nothing to save, but downloads exist. |
| Preview the generated Lua | Always. |
| See the plugins | Always; opens Plugins used (§3.7). |
| See file changes | Files change. |
| Change a section | Always; after finishing that section the user returns to the review. |
| Quit | Always. |

If validation fails, the error appears in place of the plan and save actions are not offered. Esc (or Ctrl+C) during the check cancels it.

### 4.4 Saving and result

The save screen lists completed steps and a spinner line for the active one:

```text
Saving your configuration

✓ Checked the plan and backed up current files
✓ Wrote 13 files
✓ Verified the configuration in an isolated Neovim
✓ Configuration saved
✓ Downloaded lazy.nvim
⠋ Downloading snacks.nvim  6/16

━━━━━━━━━━━━━━━───────────────────────  5 of 16 downloads
```

Downloads begin only after "Configuration saved". The result screen then shows one of:

- `✓ Your Neovim is ready`, the exact launch command (see §9.3), and — when shortcuts are configured — how to discover them with the chosen leader (`Space ?` opens the guide);
- `✓ Saved · N downloads did not finish`, with **Retry failed downloads** offered first;
- `✗ Your configuration was not saved` and the reason, which states whether previous files were restored or recovery is required.

### 4.5 Reopening

On a saved target the welcome says "Welcome back" and offers **Review & change** (recommended), Walk through every section, and Quit. Review & change opens the review with the saved state; Change a section edits one part.

### 4.6 Existing unmanaged configuration

A non-empty target not created by ShoshinVim is `read-only`. The welcome explains that it will not be changed and suggests `NVIM_APPNAME=shoshin shoshinvim` for a separate configuration. "Explore anyway" walks the sections and previews Lua; the review shows that saving is disabled. ShoshinVim never imports, merges, renames or overwrites existing Lua.

### 4.7 Recovery

A `needs recovery` target welcomes the user with the problems found and **Open recovery**. Recovery & history (also in the palette) lists every save with its time, status and changed-file count. Choosing one offers:

- Restore the files from before this save;
- Restore the files this save wrote (committed saves only; manual edits are archived in `.shoshinvim/recovery` first);
- **Cancel**, which is recommended and preselected — a destructive choice is never the default.

### 4.8 Exit

- `Ctrl+C` shows "Press Ctrl+C again to quit" (adding "your choices are not saved" when relevant); a second press quits. Any other key disarms it.
- Choosing Quit exits immediately when nothing is unsaved; otherwise it asks for the choice again.
- While files are written and verified, quitting is disabled and the notice says the save is journaled and recoverable. During downloads, `Ctrl+C` cancels the remaining downloads; the saved configuration stands.

### 4.9 Line mode

`--no-alt-screen` asks the same questions as numbered lists with normal scrollback and no escape sequences:

```text
── 2/7 Appearance ── The look of the editor.

Color scheme — The colors of the whole editor.
  1. Tokyo Night (recommended) — Deep night palette with vivid, readable syntax · downloads tokyonight.nvim
  2. Habamax — Muted dark theme · built in
Choice [1]:
```

Enter accepts the bracketed default (the saved or recommended choice), a number chooses, `b` goes back and `q` quits without writing. Multiple choice accepts space- or comma-separated numbers or `none`. Editor rows are asked one at a time. The review requires an explicit number to save. End of input with unsaved answers is an error that states nothing was written.

## 5. Keyboard model

| Context | Keys |
| --- | --- |
| Anywhere | `/` command palette · `?` keys & help · `Ctrl+C` cancel work, close an overlay, or quit when pressed twice |
| Single choice | `↑↓` or `j k` move · Enter choose · `1`–`9` choose directly · Esc, `←` or `h` back |
| Multiple choice | `↑↓` move · `space` or a number toggles · Enter continue · Esc back |
| Editor form | `↑↓` row · `←→` or `h l` change value (`space` cycles forward) · `r` recommended · Enter continue · Esc back |
| Shortcut list | `↑↓` move · Enter change the key (or continue / add on those rows) · `a` add · `d` or `x` turn off · `R` restore the style · Esc back |
| Shortcut key or command input | type in Vim notation (`/` and `?` are text) · Enter save · Esc cancel |
| Menus (welcome, review, result, recovery) | `↑↓` move · Enter choose · `1`–`9` choose directly · Esc back where a previous screen exists |
| Viewer | `↑↓`, PgUp/PgDn scroll · Esc or `q` close |
| Palette | type to filter · `↑↓` move · Enter run · Esc close |
| Set any setting | type `setting value` · Tab complete the ID · Enter set · Esc close |

`Tab`/`Shift+Tab` move like `↓`/`↑`. Every essential action is also reachable from a menu or the CLI; no action requires a mouse.

## 6. Responsive behavior

| Width × height | Behavior |
| --- | --- |
| ≥ 92 columns | Section sidebar (16 columns) beside the content. |
| 44–91 columns | Progress header (`section · position`, `step n of 8`, bar) above the content. |
| Below 44 × 14 | A resize screen shows the current and required size and that the draft is kept. **All input except `Ctrl+C` is ignored**, so no hidden decision can be answered. |

Content compresses before it scrolls (§2.4). Details truncate with `…`; summaries shorten paths with `~`. Resizing preserves the screen, cursor, draft and any open overlay. Wide screens gain whitespace, not extra panes.

## 7. Visual and content system

### Palette

A Kanagawa-inspired palette: sakura pink accent (`#D27E99` dark / `#B35B79` light), crystal blue secondary, spring green success, yellow warnings, red errors, grey for secondary text. Body text uses the terminal's own foreground. The background is queried at startup to choose light or dark accents. `NO_COLOR` or `--no-color` removes color; badges become `(recommended)`, and every state stays readable from text alone.

### Symbols

| Symbol | Meaning |
| --- | --- |
| `❯` | Selected option or row |
| `◆ recommended` | The recommended option |
| `✓` / `✗` / `–` | Done / failed / skipped |
| `●` / `○` | Current / pending section |
| `[✓]` / `[ ]` | Checked / unchecked |
| `‹ value ›` | Editable form value |
| `●●●●●●` | Color-scheme swatch |

No Nerd Font is required. The welcome shows `◯ S H O S H I N V I M` with a pink-to-blue gradient and `初心 · beginner's mind`.

### Voice

- State the outcome first; name the affected feature and path.
- Separate draft, saved, downloaded and running.
- Say "would write" before saving and "wrote" only after verification; never say "installed" from a request.
- Prefer plain words to option names; show catalog IDs only in the Lua preview, JSON view and Set any setting.
- No conversational filler or simulated personality.

## 8. State model

### State sources

| Fact | Meaning |
| --- | --- |
| Draft | In-memory decisions from this session. |
| Persisted | Last ShoshinVim-owned state written and verified. |
| Validation | Whether the draft renders and loads in an isolated Neovim. |
| Availability | Whether selected plugins and external tools are present. |
| Dirty | Whether the draft differs from the persisted state. |
| Ownership | Whether each path is managed, new, unmanaged or drifted. |

The draft is never described as saved, and requested downloads are never described as installed.

### Controller shape

```text
Model
|- target, chosen Neovim (--nvim), catalog, theme (light/dark, no-color)
|- ownership: new | saved | read-only | recovery, plus problems found
|- draft, persisted
|- screen: welcome | step | review | pick-section | working | done |
|          recovery | restore
|- position: section, question, cursor; form values; checkbox state
|- visited sections; return-to-review flag
|- plan: last validated file/action plan or its error
|- job: kind, busy, writing, progress log, spinner, cancellation
|- overlays: viewer | palette | set-setting
`- notice, quit-armed
```

Screens are rendered from this state; nothing on screen is a source of truth.

### Transaction states

```text
IDLE
  -> REVIEWING (non-mutating plan and isolated validation)
       -> BLOCKED
       -> READY
            -> LOCAL_VALIDATING -> LOCAL_BACKING_UP -> LOCAL_WRITING -> LOCAL_VERIFYING
                 -> LOCAL_FAILED_RESTORED
                 -> RECOVERY_REQUIRED
                 -> LOCAL_COMMITTED
                      -> DOWNLOADING
                           -> COMPLETE
                           -> COMPLETE_WITH_FAILED_DOWNLOADS
                           -> COMPLETE_WITH_CANCELLED_DOWNLOADS
```

Invariants:

- Local validation and writes complete before any download begins.
- Persisted state changes only at `LOCAL_COMMITTED`.
- The draft survives a local failure.
- Retrying downloads never rewrites local files.
- Unmanaged or drifted paths are never replaceable.
- Quitting is disabled during local write and verification.

## 9. Safety and ownership

### 9.1 Local writes

Before writing, ShoshinVim must:

1. validate the draft against the catalog;
2. render Lua and load it in an isolated temporary XDG environment with the chosen Neovim;
3. classify every target path by ownership and refuse unmanaged or drifted ones;
4. record a journal with backups;
5. write adjacent temporary files and rename them;
6. verify hashes and reload the generated core in isolation;
7. write state and manifest, then report the local commit separately from downloads.

The generated configuration is ordinary Lua that works without ShoshinVim. `lua/user_config/user.lua` is created once and never overwritten. Review, Preview and Check dependencies never write to the target; the preview's temporary copy is removed when Neovim closes.

### 9.2 Neovim selection

- The chosen Neovim is `--nvim`, else the `neovim` preference, else `nvim` on PATH. Every validation, probe and Mason install uses it.
- `apply` rebuilds an exported plan with the chosen Neovim and requires it to match the Neovim the plan was reviewed with. A plan naming a different Neovim is rejected **without executing the program it names**; the plan digest is a checksum, not a signature.

### 9.3 Where plugins live

Plugins and tools are stored under `shoshinvim/` in the data directory Neovim uses when it loads the target, derived from the target itself, never from the environment ShoshinVim happens to run in:

| Target | Launch command | Data directory |
| --- | --- | --- |
| `<config home>/nvim` | `nvim` | `<data home>/nvim/shoshinvim` |
| `<config home>/<name>` | `NVIM_APPNAME=<name> nvim` | `<data home>/<name>/shoshinvim` |
| Elsewhere | `nvim -u <target>/init.lua` | `<data home>/nvim/shoshinvim` |

On Windows the data folder carries Neovim's `-data` suffix, and commands use PowerShell syntax (`$env:NVIM_APPNAME='<name>'; nvim`). The result screen, line mode and the read-only hint print commands from the same rule.

## 10. Shell command surface

| Command | Behavior |
| --- | --- |
| `shoshinvim` | Open the guided setup (`--no-alt-screen` for line mode). |
| `shoshinvim --config-dir <path>` | Use an explicit target (see §9.3 for how to launch it). |
| `shoshinvim --nvim <path>` | Choose the Neovim for validation, apply and Mason. |
| `shoshinvim doctor` | Print ownership, dependencies and a safe generated-core probe. |
| `shoshinvim inspect` | Print target, ownership, state and problems. |
| `shoshinvim plan` | Validate and print or export (`--out`) the exact file/download plan; `--state` supplies declarative state, `--include-external` authorizes its downloads, `--profile` applies a scripted preset that replaces earlier values. |
| `shoshinvim apply --plan <file>` | Apply an exported plan after revalidation (§9.2); asks for confirmation unless `--yes`. |
| `shoshinvim retry` | Retry approved downloads that did not finish. |
| `shoshinvim rollback [--transaction <id>] [--generated]` | List saves, or restore one after confirmation. |
| `shoshinvim update` / `uninstall` | Act only on a receipt-owned installation; otherwise explain the owner. |
| `shoshinvim completion <shell>` / `man <dir>` | Generate completions and manual pages. |

`--output json`, `--no-color` and `--no-progress` support scripting. There is no flag that bypasses ownership, drift, target or validation checks.

## 11. Component inventory

1. Terminal lifecycle and alternate-screen shell
2. Header with status tag
3. Section sidebar and compact progress header
4. Single-choice list (numbers, pointer, badge, detail, swatches)
5. Checkbox list
6. Editor form rows
7. Menu list (welcome, review, result, recovery, section picker)
8. Review summary block
9. Progress log with spinner and download bar
10. Notice line and footer hints
11. Command palette and Set any setting input
12. Full-screen viewer (diff, dependencies, JSON, help) and the Lua preview in Neovim
13. Resize screen
14. Line-mode renderer

## 12. Acceptance criteria

The interface is acceptable when:

- a first-time user can complete setup and save by pressing Enter on recommendations, and can explain what each section chose;
- every decision shows exactly one recommendation (or a recommended set) and the cursor starts on it for a new target;
- all Neovim text and editing settings are changed in one Editor form;
- the user always knows the current section, position, and what remains;
- exact Lua, file diff, downloads and missing programs are visible before any write;
- draft, saved and downloaded states cannot be confused;
- a download failure cannot obscure a successful local save;
- an unmanaged configuration stays untouched;
- a too-small terminal cannot receive input for a hidden decision;
- everything works without color, mouse, icons or a wide terminal, and in line mode;
- the launch command shown loads the downloaded plugins;
- clean exit, cancellation, error and signal paths restore the terminal.

## 13. Usability tests

Test from an ordinary terminal with a supported Neovim. Participants may know how to open Neovim but should not need to during the task.

1. Complete setup accepting every recommendation; describe what was chosen.
2. Change tab width to 4 and restore the recommendation.
3. Choose a non-default color scheme using the swatches, then go back and change it again.
4. Deselect all languages and explain why Formatting disappeared.
5. Identify a program that must be installed manually from the review.
6. Preview the generated Lua and return without losing your place.
7. Quit from the review and confirm nothing was written.
8. Save with one simulated download failure and retry it.
9. Reopen a saved configuration and change one section from the review.
10. Open against an unmanaged configuration and confirm it stays read-only.
11. Recover from simulated drift without losing manual content.
12. Finish the setup in an 80×24 terminal and in line mode.

Record wrong turns, hesitation at recommendations, understanding of status tags, and time to confidence.

## 14. Design references

Interaction references, refreshed on 2026-09-28:

- [Grok Build overview](https://docs.x.ai/build/overview) — fullscreen terminal experience and restrained status display.
- [Grok Build keyboard shortcuts](https://docs.x.ai/build/keyboard-shortcuts) — cancellation and terminal fallbacks.

These products are references, not dependencies. ShoshinVim adopts their look and keyboard grammar but not their conversational transcript; a configuration generator is clearer as guided decisions than as chat.
