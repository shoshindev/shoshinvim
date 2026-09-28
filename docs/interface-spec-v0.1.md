# ShoshinVim v0.1 interface specification

> Staged implementation: see [current alpha coverage and release gates](implementation-status.md). This document specifies the shipped guided-section interface. The [configuration catalog](configuration-catalog.md) also lists planned capabilities; a listed plugin or installation channel does not imply released support.

- Status: authoritative interaction, safety, state, and responsive-design specification
- Audience: the developer implementing the standalone terminal application
- Product line: **Build a Neovim you understand, one decision at a time.**
- Feature scope: [configuration catalog](configuration-catalog.md)
- Interface direction: keyboard-first terminal UI with the feel of Claude Code and Grok Build — selectable menus, quiet hints, a command palette — without a chat transcript
- Implementation: `internal/ui` (`steps.go` holds every section, question and recommendation; `model.go`, `view.go` and `jobs.go` hold state, rendering and background work; `plain.go` is line mode)

This document specifies behavior and presentation, not production code. ShoshinVim is a deterministic local application. It borrows the look and keyboard grammar of modern coding-agent TUIs without requiring an AI model, cloud service, account, or autonomous tool use.

## 1. Product direction

### Primary interaction model

Running `shoshinvim` opens a fullscreen guided setup:

1. A welcome panel states what will happen and that nothing is written before review.
2. Seven sections — **Editor, Appearance, Workspace, Navigation, Languages, Formatting, Git & tools** — each ask a short series of decisions.
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

- **Command palette** (`/`): a panel with a filter input and the command list.
- **Set any setting**: an input accepting `setting value`, with up to six ID suggestions and Tab completion.

Full-screen viewers show long content with a title, scroll percentage and `esc` to close: generated Lua, file diff (colored), dependencies, all settings as JSON, and keys & help.

## 3. Setup structure

### 3.1 Question kinds

| Kind | Used for | Interaction |
| --- | --- | --- |
| Single choice | Most decisions | `↑↓` or a number selects; Enter or the number confirms. |
| Multiple choice | Languages | `space` or a number toggles `[✓]`; Enter continues. |
| Form row | Editor settings | All rows on one screen; `←→` cycles a row's value; Enter continues. |

Every option sets explicit catalog values, so the saved state records each decision even when it matches Neovim's default.

### 3.2 Recommendations

- Every single-choice decision and form row has exactly one recommended option; a multiple-choice decision states its recommended set.
- On a new target the cursor (or form value) starts at the recommendation. On a saved target it starts at the saved choice; a value set outside the curated options displays as `custom`.
- Options show `◆ recommended` (or `(recommended)` without color). In the Editor form, a row that differs shows `changed · recommended: <value>`, and `r` restores the recommendation.
- Recommendations never lock alternatives and never overwrite a saved choice.

### 3.3 Sections and recommended choices

| Section | Decisions (recommended first) |
| --- | --- |
| **Editor** (form, 19 rows) | Line numbers on · Relative numbers off · Wrap off · Indent with spaces · Tab width 2 · Indent width 2 · Search case smart · Replace preview split · Context lines 8 · Context columns 8 · Persistent undo on · Unsaved changes ask · New splits right & below · Mouse all modes · Clipboard shared · Cursor line highlight · Invisible characters trailing spaces · Sign column always · Key timing fast (400 ms / 250 ms) |
| **Appearance** | Color scheme: Tokyo Night / Habamax / Sorbet / Retrobox / Quiet / Default (with color swatches) · Status line: Lualine / built in · Open files: tabs across the top / no tabs · Popup borders: rounded / square / none |
| **Workspace** | Keyboard shortcuts: Space shortcuts with start screen and guide / Neovim defaults · Shortcut hints: show / none |
| **Navigation** | Find files and text: Snacks picker / mini.pick / built in · File explorer: project sidebar / Oil / netrw |
| **Languages** | Which languages: TypeScript & JavaScript, HTML, CSS, JSON, Lua (all recommended) · Completion: as you type / off · Errors and warnings: inline / signs only · Server settings: written by ShoshinVim / add nvim-lspconfig |
| **Formatting** | Formatter: Prettier & StyLua / language server · When to format: when I ask / every save |
| **Git & tools** | Git changes: markers in the gutter / off · Language tools: install them for me (Mason) / I'll install them |

Each option carries a short detail line naming its effect and any download (for example "downloads lualine.nvim" or "built in").

### 3.4 Skipped decisions

When the user has decided on languages and selected none, completion, diagnostics, server settings, both Formatting decisions and the language-tools decision are skipped, and their values are removed from the draft. Skipped sections are omitted from navigation and summarized as "skipped".

### 3.5 Derived settings

Implementation details follow from decisions and are never asked:

- `plugins.manager` is lazy.nvim exactly when a chosen option needs a plugin, otherwise none.
- Lualine sets `showmode` off and a global status line (`laststatus` 3); the built-in status line keeps Neovim's defaults.
- Completion as you type sets `completeopt` and `pumheight`.
- `termguicolors` is left to Neovim's own terminal detection.

### 3.6 Summaries

The review and "Change a section" list summarize each section on one line using short option names, for example `Tokyo Night · Lualine · Buffer tabs · Rounded borders`. The Editor summary reports `19 settings · all recommended` or the number changed.

## 4. Core flows

### 4.1 First launch

The welcome panel shows the logo, the tagline, the seven section names, the recommendation convention and "Nothing is written or downloaded until you review and save." Below it: the target and status tag, then **Start setup** (recommended) and Quit.

### 4.2 Walking the sections

Enter on a choice records it in the in-memory draft and moves to the next decision; the last decision of a section moves to the next section, and the last section opens the review. Esc goes back one decision — across section boundaries, and from the first decision to the welcome screen. Answers are kept when going back. Leaving the Editor form with Enter or Esc keeps its rows.

### 4.3 Review

Opening the review builds a fresh, non-mutating plan in the background ("Checking your configuration with Neovim…"), which validates generated Lua in an isolated Neovim. The screen shows:

- one summary line per section;
- **Files**: new, updated and removed counts and the target;
- **Downloads**: plugin and language-tool counts;
- **Install first**: programs the user must provide (for example ripgrep or Node.js). Tools that Mason will download are not listed.

Then the actions, in order and as applicable:

| Action | When |
| --- | --- |
| **Save and download** (recommended) | Files change and downloads exist. |
| Save without downloading | Files change and downloads exist; missing downloads stay pending. |
| **Save** (recommended) | Files change and nothing needs downloading. |
| **Download plugins and tools** (recommended) | Nothing to save, but downloads exist. |
| Preview the generated Lua | Always. |
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

- `✓ Your Neovim is ready`, the exact launch command (see §9.3), and — with Space shortcuts — how to discover them;
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

The generated configuration is ordinary Lua that works without ShoshinVim. `lua/user_config/user.lua` is created once and never overwritten. Review, Preview and Check dependencies never write.

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

On Windows the data folder carries Neovim's `-data` suffix. The result screen and line mode print the launch command from the same rule.

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
12. Full-screen viewer (Lua, diff, dependencies, JSON, help)
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

- [Claude Code interactive mode](https://code.claude.com/docs/en/interactive-mode) — keyboard-first input, selectable prompts, command discovery.
- [Claude Code commands](https://code.claude.com/docs/en/commands) — slash-command palette and focused viewers.
- [Grok Build overview](https://docs.x.ai/build/overview) — fullscreen terminal experience and restrained status display.
- [Grok Build keyboard shortcuts](https://docs.x.ai/build/keyboard-shortcuts) — cancellation and terminal fallbacks.

These products are references, not dependencies. ShoshinVim adopts their look and keyboard grammar but not their conversational transcript; a configuration generator is clearer as guided decisions than as chat.
