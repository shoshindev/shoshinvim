# ShoshinVim v0.1 interface specification

> Staged implementation: see [current alpha coverage and release gates](implementation-status.md). This specification includes planned capabilities; a listed plugin or installation channel does not imply released support.

- Status: authoritative interaction, safety, state, and responsive-design specification
- Audience: the developer implementing the standalone terminal application
- Product line: **Build your Neovim from the terminal.**
- Feature scope: [configuration catalog](configuration-catalog.md)
- Interface direction: conversational terminal UI inspired by Grok Build and Claude Code

This document specifies behavior and presentation, not production code. ShoshinVim is a deterministic local application. It borrows the interaction grammar of modern coding-agent TUIs without requiring an AI model, cloud service, token meter, agent sessions, or autonomous tool use.

## 1. Product direction

### Primary interaction model

Running `shoshinvim` from a shell opens a fullscreen terminal conversation:

1. A scrollable transcript occupies most of the screen.
2. A compact composer at the bottom accepts slash commands and answers to the current question.
3. A one-line status bar shows configuration state and context.
4. A short footer exposes the keys that matter now.
5. Pickers, confirmations, help, generated Lua, and diffs appear only when requested.

The interface should feel closer to Grok Build or Claude Code than to a settings dashboard. The user gives an intent, the configurator replies with a concrete proposal, and the user can inspect or revise it before saving.

ShoshinVim is a standalone executable. It may call `nvim --headless` for validation and health probes, but the user does not need to start Neovim to configure it. The application owns the terminal alternate screen, restores terminal state on every exit path, and offers `--no-alt-screen` for constrained terminals and captured logs.

### Patterns intentionally adopted

- Transcript plus persistent bottom input.
- Slash commands discoverable from `/` and a command palette.
- One question or decision at a time during guided setup.
- Inline summaries for work, warnings, progress, and results.
- Expandable detail instead of permanently visible side panels.
- A compact status line for mode, draft state, health, and restart state.
- Clear cancellation and interruption behavior.
- Keyboard-first operation with optional mouse support.
- Fullscreen viewers for long content such as Lua and diffs.

### Patterns intentionally not adopted

- Model selection, token/context usage, cost, or reasoning controls.
- Chat sessions, forks, subagents, background agents, or task dashboards.
- Free-form AI interpretation of configuration requests.
- Autonomous shell execution or silent dependency installation.
- Persistent navigation sidebars, card grids, dashboard tiles, or wizard steppers.
- A permanent details pane or three-column layout.
- Decorative chat bubbles, avatars, or simulated typing.

## 2. Interface anatomy

### 2.1 Terminal application shell

The application detects terminal capabilities, target configuration directory, platform, and available Neovim executable before rendering. Exiting restores the cursor, screen mode, and input state. A crash handler makes a best effort to restore the terminal before printing a plain-text recovery message.

The fullscreen shell contains:

```text
+--------------------------------------------------------------+
| ShoshinVim                                  ~/.config/nvim     |
+--------------------------------------------------------------+
|                                                              |
|  transcript: prompts, replies, proposals, status, results    |
|                                                              |
|                                                              |
+--------------------------------------------------------------+
| > /                                                          |
+--------------------------------------------------------------+
| DRAFT  3 changes  health: 1 missing       Tab focus  ? help   |
+--------------------------------------------------------------+
```

This is a structural diagram, not a border requirement. On small terminals, visual separators may collapse to one line.

### 2.2 Transcript

The transcript is application-owned scrollback rendered as terminal text. Copy mode exposes plain text, and `--no-alt-screen` emits readable output into the ordinary terminal history. Search, copying, screen readers, and low-color environments must not depend on decorative styling.

Every entry has a stable ID and one of these roles:

- `user` — submitted command or answer;
- `configurator` — explanation, question, or summary;
- `proposal` — a group of draft changes;
- `choice` — the currently answerable options;
- `activity` — validation, write, or installation progress;
- `result` — completed local or external work;
- `warning` — non-blocking risk or missing optional capability;
- `error` — blocked action or failed operation.

New output appends to the transcript. It never clears earlier decisions merely because the user navigates elsewhere. `/clear` may start a visually clean transcript only when there is no unsaved draft; it does not erase persisted state or recovery records.

Long entries render a short summary plus an explicit expansion action. Critical warnings, affected paths, confirmation choices, and final status must never be hidden only inside collapsed detail.

### 2.3 Composer

The composer is one line by default:

```text
> /set editor.wrap off
```

It accepts:

- slash commands;
- a number or exact label for an active choice;
- `yes` or `no` only when a confirmation explicitly requests it;
- search text while a picker is active.

The initial product does not claim to understand unrestricted natural language. Unknown text produces a short correction and matching commands rather than guessing.

The composer retains unsent text when focus moves to the transcript or a viewer. Prompt history is local to the configurator and excludes secrets because the product never asks for credentials.

### 2.4 Status line

The status line is concise and factual. It may contain:

```text
DRAFT | 3 changes | health: 1 missing | restart: no | editor.wrap
```

Allowed state labels:

- `NEW` — no managed configuration exists;
- `SAVED` — draft equals persisted configuration;
- `DRAFT` — unsaved changes exist;
- `BLOCKED` — local save is unsafe or invalid;
- `SAVING` — the local transaction is active;
- `INSTALLING` — reviewed external work is active after local commit;
- `RECOVERY` — an incomplete or drifted transaction requires attention;
- `READ ONLY` — an unmanaged existing configuration was detected.

Color may reinforce these labels but never replace them.

### 2.5 Contextual footer

The footer lists no more than four actions and changes with focus. Examples:

```text
Enter send   Tab transcript   Ctrl+P commands   ? help
```

```text
j/k move   Enter expand   y copy   Tab prompt
```

The full key reference lives in `?` and `/help`; it is not permanently printed.

### 2.6 Transient surfaces

Use a centered overlay or temporary full-screen page only for:

- command palette;
- choice/search picker;
- confirmation;
- help and key reference;
- a focused blocking error.

Use full-screen TUI pages for:

- generated Lua;
- unified or side-by-side diff;
- long file plans;
- health details;
- recovery comparisons.

Generated Lua, diffs, and reports can also be opened in `$PAGER` or `$EDITOR` only after an explicit action. Returning from a transient surface restores both composer content and transcript selection.

## 3. Conversation grammar

### 3.1 Core commands

| Command | Purpose |
| --- | --- |
| `/start` | Start or restart guided configuration without changing files. |
| `/back` | Return to the preceding guided question or conversation context. |
| `/skip` | Leave an optional guided question unchanged and continue. |
| `/why` | Explain the active question and its generated effect. |
| `/profile [name]` | Inspect or select Balanced IDE, Minimal native, Full IDE, Writing, Remote, Language-focused, or Custom. |
| `/configure [topic]` | Ask guided questions for one catalog topic. |
| `/compare <capability>` | Compare the recommended choice with supported alternatives and tradeoffs. |
| `/set <id> <value>` | Change one validated setting. |
| `/enable <id>` | Enable one feature and show implications. |
| `/disable <id>` | Disable one feature and show implications. |
| `/show [topic]` | Summarize the current draft or one topic. |
| `/search <text>` | Search setting, feature, plugin, tool, and command names. |
| `/undo` | Reverse the latest configurator draft mutation. |
| `/reset [topic|all]` | Preview and confirm a draft reset. |
| `/review` | Validate and show the complete proposed change. |
| `/lua [setting|file]` | Open generated Lua for the draft. |
| `/diff` | Open persisted-versus-draft or recovery diff. |
| `/files` | Inspect exact paths and ownership. |
| `/health [topic]` | Recheck non-mutating dependency evidence. |
| `/save` | Begin preflight and reviewed save confirmation. |
| `/retry` | Retry only failed external acquisition work. |
| `/recovery` | Inspect drift, journals, backups, and safe actions. |
| `/clear` | Start a clean transcript only when no unsaved draft would be hidden. |
| `/help [topic]` | Explain commands, keys, states, or a catalog item. |
| `/quit` | Close safely or ask what to do with an unsaved draft. |

Entering `/` opens completion. Commands remain available even while a guided question is active. Commands that would abandon the active question say so and require confirmation only when information would actually be lost.

### 3.2 Stable catalog IDs

Commands address settings and features through stable dot-separated IDs such as:

```text
editor.line_numbers
editor.wrap
appearance.colorscheme
search.telescope
language.lua.lsp
format.lua.stylua
debug.go
```

Completion shows the beginner-facing label beside the ID. The transcript normally uses the friendly label; `/show --technical` and generated-code annotations expose IDs.

### 3.3 Guided questions

A guided question contains:

1. one plain-language question;
2. two to seven numbered choices;
3. the proposed or current value;
4. implications, if any;
5. `Enter` behavior and a visible skip/back option when valid.

Example:

```text
How should long lines behave?

  1. Wrap at the window edge
  2. Keep each line on one visual row  [Recommended for code]

Current proposal: Keep each line on one visual row
Reply with 1 or 2.  /why explains this choice.  /back returns.
```

Choosing a value appends a compact result rather than repainting the old question as if history changed:

```text
Changed Wrap long lines: On -> Off
```

When a capability has several good implementations, the question shows all curated choices, one context-specific recommendation, and the built-in/none option when valid:

```text
Choose a fuzzy finder:

  1. Snacks picker     [Recommended: balanced]
  2. fzf-lua           [Recommended: minimal]
  3. Telescope         [Alternative: extension ecosystem]
  4. mini.pick         [Alternative: compact]
  5. Native commands   [Built in]

Type /compare navigation.picker for requirements and tradeoffs.
```

A recommendation mark explains why it applies to the active profile. It never disables alternatives, hides dependencies, or changes an existing manual choice after a catalog refresh.

### 3.4 Proposals

Profiles and features are proposals, not opaque presets. A proposal entry shows:

- what was requested;
- settings and features that would change;
- new plugins or external tools implied;
- missing dependencies;
- whether existing manual choices are preserved;
- commands to accept, inspect, or cancel.

Accepting a proposal changes only the in-memory draft. It never writes files or installs tools.

### 3.5 Shell command surface

The interactive slash commands have non-interactive CLI counterparts where safe:

| Shell command | Behavior |
| --- | --- |
| `shoshinvim` | Open the interactive fullscreen TUI. |
| `shoshinvim --config-dir <path>` | Open against an explicit target instead of the platform default. |
| `shoshinvim doctor` | Print dependency, ownership, and generated-config health without mutation. |
| `shoshinvim inspect` | Print discovered target, state, manifest, plugins, and tools. |
| `shoshinvim plan` | Validate the draft and print or export the exact file/external-action plan. |
| `shoshinvim apply --plan <file>` | Apply a previously exported plan after hashes and target identity still match. |
| `shoshinvim rollback` | Open an interactive recovery choice; never select a destructive default. |
| `shoshinvim update` | Explain the installed update channel and invoke its supported updater. |
| `shoshinvim uninstall` | Remove the application through its install receipt; generated Neovim config remains. |

Machine-readable commands support `--output json`, documented exit codes, `--no-color`, and `--no-progress`. `apply` requires an interactive confirmation unless both `--yes` and the exact exported plan are supplied. There is no flag that bypasses ownership, drift, target, or validation checks.

## 4. Core flows

### 4.1 First launch

The opening transcript is short:

```text
ShoshinVim
Build a readable Neovim configuration from the terminal and review every file before it changes.

No managed configuration was found at ~/.config/nvim.

Choose a starting profile:
  1. Balanced IDE  [Recommended]
  2. Minimal native
  3. Full IDE
  4. Writing and notes
  5. Remote and low-resource
  6. Language-focused
  7. Custom

Reply with a number, or type /help profiles.
```

After profile selection, the configurator summarizes the proposal and asks which topic to refine. It does not force the user through every catalog section. `/review` is available as soon as the draft is valid.

### 4.2 Reopening

When managed state exists, the transcript starts with:

- saved configuration status;
- draft status, normally none;
- runtime differences or restart need;
- missing selected dependencies;
- the most useful next commands.

There is no Overview dashboard. `/show`, `/health`, `/configure`, and `/review` provide the same information through the transcript and focused viewers.

### 4.3 Editing one topic

`/configure core` begins a bounded sequence of editor questions. The user may answer, skip, go back, search, inspect generated Lua, or leave the sequence without losing accepted draft changes.

After each answer, validation runs only for affected state. The configurator reports a concise change and continues. It does not open a separate settings screen.

### 4.4 Review

`/review` performs a fresh, non-mutating preflight and appends:

```text
Review ready

  Draft                 7 changes
  Files                 5 create, 0 modify
  Plugins               19 requested
  External tools        4 present, 2 missing
  Local write           READY WITH PENDING FEATURES

  /show changes         inspect decisions
  /files                inspect exact paths
  /lua                  inspect generated Lua
  /diff                 compare with persisted output
  /save                 continue to confirmation
```

Warnings name the exact affected feature. “Missing dependency” never becomes a generic failure when local generation remains safe.

### 4.5 Save confirmation

`/save` never writes immediately. It refreshes validation and displays:

- exact paths created or modified;
- backup paths;
- ownership status;
- local changes;
- separate plugin/tool downloads and build commands;
- unresolved warnings;
- what rollback can and cannot undo.

The final prompt requires an explicit choice:

```text
Proceed with the reviewed local write?
  1. Save local configuration
  2. Save, then run the listed external actions
  3. Cancel
```

Option 2 appears only when external actions were listed in Review. Confirmation cannot be inferred from Enter on a previous screen.

### 4.6 Progress and result

Progress is appended as a single activity block whose lines update in place while active:

```text
Saving local configuration
  [OK] validated generated Lua
  [OK] created backup
  [..] writing 3 of 5 files
  [  ] verifying hashes
```

The final block is immutable transcript history:

```text
Local configuration saved and verified.
5 files written. Backup: ...
Restart Neovim to load 3 changes.
```

External work starts only after the local commit and receives a separate block and outcome. A plugin download failure cannot relabel a verified local save as failed.

### 4.7 Existing unmanaged configuration

When the target config is non-empty and not owned by the configurator:

- status is `READ ONLY`;
- the transcript explains why writing is disabled;
- `/show`, `/health`, and catalog exploration remain available;
- `/save` explains the block;
- the product does not import, merge, rename, or overwrite existing Lua.

### 4.8 Drift and recovery

Manual changes to owned generated files or an incomplete transaction change status to `RECOVERY`. Normal save is blocked. `/recovery` opens a focused comparison and offers only evidence-backed actions:

- cancel and leave files untouched;
- restore a verified backup;
- restore the last generated version;
- copy or move manual changes into the user-owned file, with exact paths shown.

No default action is preselected when it could discard work.

### 4.9 Exit

With no draft and no active transaction, `/quit`, `Ctrl+D` on an empty composer, or the terminal's normal close request exits and restores terminal state.

With an unsaved draft, the configurator asks:

```text
You have 7 unsaved configuration changes.
  1. Return to the session
  2. Discard the draft and quit
```

Save is not offered as an ambiguous one-key exit action; the user returns and runs `/review` or `/save`.

Close is blocked during the local write/verify phase. After local commit, external work may be safely cancelled or left pending according to the adapter's verified behavior.

## 5. Keyboard model

### Composer focus

| Key | Action |
| --- | --- |
| `Enter` | Submit command or answer. |
| `Tab` | Complete command or move focus to transcript when completion is closed. |
| `Shift-Tab` | Move focus to transcript. |
| `Up` / `Down` | Previous or next prompt-history entry. |
| `Ctrl-R` | Search prompt history. |
| `Ctrl-P` | Open command palette. |
| `Ctrl-C` / `Esc` | Cancel completion, picker, or active non-write work; clear input when idle. |
| `Ctrl-D` | Quit only when input is empty and exit is safe. |
| `?` | Open help only when the composer is empty. |

### Transcript focus

| Key | Action |
| --- | --- |
| `j` / `Down` | Select next entry or action. |
| `k` / `Up` | Select previous entry or action. |
| `Ctrl-D` / `Ctrl-U` | Scroll half a page. |
| `PageDown` / `PageUp` | Scroll one page. |
| `g` / `G` | First or latest entry. |
| `Enter` | Expand, collapse, or invoke the selected action. |
| `y` | Copy selected entry as plain text. |
| `/` | Search transcript text. |
| `Tab` / `i` | Return to composer. |
| `?` | Open contextual help. |

Key handling is scoped to the running TUI. Terminal limitations are documented in help, and every essential action has a slash-command or shell-command alternative.

## 6. Responsive behavior

Width modes are based on the current terminal viewport.

### Wide: 120 columns or more

- Transcript remains a single readable column capped at a comfortable text width.
- Review summaries may use aligned two-column values.
- Diffs may open side by side when each side remains readable.
- Extra width is whitespace, not a permanent sidebar.

### Normal: 80–119 columns

- Transcript uses the available width.
- Review uses stacked sections.
- Diffs default to unified format.

### Narrow: 52–79 columns

- Header reduces to the product name.
- Status values wrap to a second line if necessary.
- Choice descriptions stack below their number.
- Footer shows two actions and points to `?` for the rest.
- Long paths abbreviate in summaries but remain fully available in detail.

### Below 52 columns or 16 rows

Render a resize blocker containing current size, required minimum, `/quit`, and a reminder that the draft is still in memory. Do not squeeze confirmations or truncate paths into an unsafe layout.

Resizing preserves selected transcript entry, composer text, active question, and scroll position as closely as possible.

## 7. Visual and content system

### Text hierarchy

Use spacing and concise labels rather than boxes around every item. Borders separate only major regions or transient dialogs.

Required literal states:

- `[OK]`
- `[WARN]`
- `[MISSING]`
- `[ERROR]`
- `[BLOCKED]`
- `[RESTART]`
- `[PENDING]`

ASCII equivalents are canonical. Nerd Font icons may be an optional enhancement but never the sole signal.

### Voice

- State the outcome first.
- Name the affected feature and path.
- Separate selected, saved, installed, and active.
- Avoid conversational filler and simulated personality.
- Say “will write” before confirmation and “wrote” only after verification.
- Never say “installed” from request state alone.

### Explanations

Every setting or feature supports:

- one-sentence purpose;
- current draft value;
- recommendation context and last-reviewed evidence;
- generated Lua mapping;
- plugin and external-tool implications;
- runtime/restart behavior;
- exact reason for warnings or blocks.

`/why` explains the active question. `/help <id>` explains any catalog item.

## 8. State model

### State sources

Keep these facts separate:

| Fact | Meaning |
| --- | --- |
| Draft | In-memory proposal edited through this session. |
| Persisted | Last configurator-owned state successfully written and verified. |
| Probed runtime | Value observed by an explicit isolated `nvim --headless` validation or health probe. |
| Validation | Whether the draft can be rendered and written safely. |
| Availability | Whether selected plugins and external tools are present. |
| Dirty | Whether draft differs from persisted state. |
| Restart | Whether a saved change requires reopening or restarting any already-running Neovim session. |
| Ownership | Whether each path is managed, new, unmanaged, drifted, or unknown. |

Draft is never described as saved. Persisted is never described as runtime-verified without probe evidence. Requested external work is never described as installed.

### Controller shape

```text
AppState
|- lifecycle: opening | ready | closing
|- surface: conversation | viewer | overlay | resize_blocker
|- status: new | saved | draft | blocked | saving | installing |
|           recovery | readonly
|- environment: platform, terminal capabilities, cwd, target and Neovim executable
|- transcript: ordered entries with stable IDs and expansion state
|- composer: text, history, completion, focus
|- prompt: none | choice | confirmation | input
|- viewer: none | lua | diff | files | health | recovery | help
|- persisted: state and manifest snapshot or absent
|- draft: profile provenance, explicit values, feature choices
|- history: undoable draft mutations
|- validation: per-item and aggregate results
|- dependencies: check evidence and timestamps
|- files: ownership and proposed file plan
|- transaction: local and external phase state
`- notifications: durable warnings plus current ephemeral message
```

The transcript is rendered from state; it is not the source of product truth.

### Readiness

- `BLOCKED` — local configuration cannot safely be written.
- `READY WITH PENDING FEATURES` — local write is safe, but selected external capabilities are missing.
- `READY` — local preflight passes and currently checked prerequisites are present.

`READY` permits confirmation; it does not mean the save has succeeded.

### Transaction states

```text
IDLE
  -> PREFLIGHT
       -> BLOCKED
       -> CONFIRMATION
            -> LOCAL_VALIDATING
            -> LOCAL_BACKING_UP
            -> LOCAL_WRITING
            -> LOCAL_VERIFYING
                 -> LOCAL_FAILED_RESTORED
                 -> RECOVERY_REQUIRED
                 -> LOCAL_COMMITTED
                      -> EXTERNAL_INSTALLING
                           -> COMPLETE
                           -> COMPLETE_WITH_PENDING_FEATURES
                           -> COMPLETE_WITH_EXTERNAL_FAILURES
```

Invariants:

- Local validation and writes complete before external acquisition begins.
- Persisted state changes only at `LOCAL_COMMITTED`.
- The draft survives local failure.
- External retry cannot re-enter local writing.
- Unmanaged or drifted paths are never treated as replaceable.
- Closing is blocked during local write and verification.

## 9. Safety and ownership

The interface must make the underlying generator safer, not merely friendlier.

Before a local write it must:

1. resolve and validate the draft;
2. render and compile Lua in memory;
3. validate the rendered tree in an isolated temporary XDG environment with the selected Neovim executable;
4. classify every target path by ownership;
5. show the exact file plan and diff;
6. create a recoverable journal and backups;
7. write adjacent temporary files and rename atomically;
8. write state and manifest data;
9. verify final hashes and rerun the safe headless probe;
10. report local commit separately from external work.

The generated configuration remains ordinary Lua and works without the configurator installed. A user-owned extension file is created once and never overwritten.

`/health` and `/review` are non-mutating. Opening instructions, copying a command, or requesting an installation never counts as successful installation.

## 10. Component inventory

The v0.1 interface needs only these reusable components:

1. terminal lifecycle and alternate-screen shell;
2. scrollback transcript;
3. prompt/composer editor;
4. status line and contextual footer;
5. transcript entry renderer;
6. numbered-choice prompt;
7. command completion/palette;
8. confirmation overlay;
9. full-screen text viewer;
10. generated-Lua viewer;
11. diff viewer;
12. progress/result entry;
13. resize blocker;
14. help view.

There is no sidebar, dashboard card, wizard stepper, settings table, persistent context panel, or separate settings landing screen.

## 11. Acceptance criteria

The conversational shell is acceptable when:

- a first-time user can select a profile, change one setting, review, and save without learning a navigation hierarchy;
- the complete flow starts from an ordinary shell without first opening Neovim;
- every catalog item is reachable by command completion and search;
- the current question and safe next action remain obvious after scrolling;
- the user can inspect exact Lua, paths, and diff before confirmation;
- draft, saved, installed, runtime-active, and restart states cannot be confused;
- all essential behavior works without color, mouse, icons, or a wide terminal;
- an external failure cannot obscure a successful local commit;
- an existing unmanaged configuration remains untouched;
- resize and viewer transitions preserve draft and composer state;
- no unknown free-form input silently mutates configuration.
- clean exit, cancellation, error, panic, and signal paths restore terminal state.

## 12. Usability tests

Test from an ordinary terminal with a supported `nvim` executable available. Participants may know how to open and quit Neovim but should not need to open it during the task.

1. Start a Web profile and explain what it will add.
2. Disable line wrapping with command completion.
3. Compare at least three implementations of one capability and choose a non-default alternative.
4. Identify a missing tool and the exact affected capability.
5. Inspect generated Lua and return without losing prompt text.
6. Predict every file changed by `/save`.
7. Cancel at final confirmation and confirm no disk changes.
8. Complete a save with one simulated external failure.
9. Reopen and change one persisted value.
10. Attempt to quit with an unsaved draft.
11. Open against an unmanaged config and verify it remains read only.
12. Recover from simulated drift without losing manual content.

Record command discovery, wrong turns, interpretation of status labels, confirmation comprehension, and time to confidence.

## 13. Design references

The interaction direction was refreshed on 2026-09-28 from official documentation:

- [Grok Build overview](https://docs.x.ai/build/overview) — fullscreen interactive TUI and status-line-led terminal experience.
- [Grok Build keyboard shortcuts](https://docs.x.ai/build/keyboard-shortcuts) — prompt/scrollback focus, command discovery, cancellation, and terminal-specific fallbacks.
- [Grok Build status line](https://docs.x.ai/build/features/status-line) — compact live context near the prompt.
- [Claude Code interactive mode](https://code.claude.com/docs/en/interactive-mode) — keyboard-first input, command completion, history, transcript navigation, and diff review.
- [Claude Code commands](https://code.claude.com/docs/en/commands) — slash-command discovery and focused viewers.

These products are references, not dependencies. ShoshinVim uses its own terminology and only the interaction patterns appropriate to a deterministic configuration generator.
