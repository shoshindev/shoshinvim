# nvim-configurator v0.1 interface specification

- Status: interaction, safety, state, and responsive-design specification
- Audience: the developer implementing the Lua interface
- Product line: **Build your Neovim from inside Neovim.**
- Authoritative feature scope: [configuration catalog](configuration-catalog.md)

This document specifies behavior, layout, state, safety, and language. It does not specify production Lua implementation.

## Scope update

The configuration catalog now targets complete, progressively presented coverage of the current `nvim-lua/kickstart.nvim` defaults and optional examples. The original narrow JavaScript/TypeScript plus `mini.pick` flow in this document remains a useful vertical-slice prototype, but it is no longer the complete product scope.

When this document conflicts with `docs/configuration-catalog.md` about available settings, plugins, languages, dependencies, generated modules, navigation sections, or deferred features, the catalog wins. The ownership model, state distinctions, preview-before-save behavior, transactional writes, recovery design, keyboard accessibility, and responsive layout in this document remain authoritative.

## 1. Interface design summary

### Primary interaction model

`nvim-configurator` should open a dedicated tab page containing a full-screen, read-only scratch buffer. The buffer acts like a native Neovim configuration document: the cursor moves through real lines, actions are buffer-local mappings, long content scrolls normally, and generated code opens in normal Neovim windows. The configurator remembers the tab and window from which it was opened and returns there when it closes.

This is preferable to a large floating window for five reasons:

1. A full tab survives narrow terminals and long explanations better.
2. Normal Neovim scrolling, splits, search, diff, and help conventions remain available.
3. A beginner can understand that the configurator is a temporary workspace, not a desktop control panel.
4. Long paths, generated Lua, and error details need real buffer space.
5. Focus and resize behavior are substantially simpler for a one-developer implementation.

Small floating windows are appropriate only for a focused interruption: a setting explanation, a two- or three-choice confirmation, short technical details, or a search/input field. No primary screen should be a large float.

No external UI library is justified for v0.1. Native buffers, windows, extmarks, highlight namespaces, `vim.ui.input` where a short text query is useful, and native diff facilities cover the scope. An external UI dependency would be especially costly in a tool whose job is to help a fresh installation acquire its first dependencies.

### Experience model

The interface has two forms:

- **First-run onboarding:** a short, ordered path with one primary decision per screen.
- **Later settings:** a browsable configuration workspace opened with `:NvimConfig`; `:Settings` is an optional guarded alias.

The onboarding sequence is:

1. Welcome
2. Profile
3. Editor essentials
4. Development tools
5. Optional features
6. Review and save

The ordinary path stays short because a profile proposes the complete Kickstart-compatible state. Editor essentials shows the most consequential native choices. Development tools and Optional features show selected summaries with an explicit “Customize” action rather than forcing the user through every catalog row. Generated Lua, dependency guidance, and diff are subordinate views, not additional mandatory steps. A beginner accepting a profile should normally finish in five to eight minutes; a fully custom setup has no artificial time target.

The UI never applies draft values live. A changed row means “proposed,” not “active.” Saving writes and validates files; a restart makes the configuration the startup configuration. This conservative rule gives every visible state one unambiguous meaning and avoids a half-applied current session.

### Safety model

The configurator owns only files it created and recorded in its manifest. It distinguishes four path categories:

- **Configuration files:** under the expanded `stdpath("config")`, shown to the user as absolute paths.
- **Neovim plugin artifacts:** `nvim-pack-lock.json` under the config path and plugin directories under the expanded built-in `vim.pack` data location.
- **Configurator records:** manifest, transaction journal, and diagnostic data under the expanded `stdpath("state")`.
- **Backups:** timestamped copies under the configurator state directory.

The recommended generated configuration layout is modular and selection-driven:

- `init.lua`
- focused modules under `lua/user_config/` for editor, appearance, diagnostics, keymaps, plugins, search, Git, languages, completion, formatting, linting, Treesitter, and debugging
- `lua/user_config/user.lua`, created once and never overwritten
- `nvim-pack-lock.json` when Neovim's built-in plugin manager records plugins

Only selected groups are emitted. The authoritative module list and ownership policy are defined in the configuration catalog.

The generated files must not require the configurator at startup. Their comments should say that they are ordinary Lua and remain active after the configurator is removed.

Before any write, the controller validates ownership, file hashes, generated Lua, target permissions, and known dependency state. It writes a transaction journal before mutating the target, creates a recoverable backup when an earlier generated configuration exists, writes through temporary sibling files, verifies the final files, and only then marks the local configuration saved.

Plugin acquisition is a second phase. “Configuration saved and verified” and “plugin installed” are separate results. A plugin download failure must never be described as though the local configuration write also failed.

### Plugin-management recommendation

Use Neovim 0.12's built-in `vim.pack` capability rather than adding a third-party plugin manager. In the beginner-facing UI, call it “Neovim's built-in plugin installer.” Put the technical name and its 0.12 experimental status in details. The Review screen must disclose that Git is used, show the resolved plugin directory, and show that `nvim-pack-lock.json` may change.

Generated plugin setup must be failure-tolerant: a missing optional plugin may remove that feature, but it must not prevent the rest of the generated configuration from loading.

### Beginner-facing vocabulary

Primary labels use outcomes; technical terms live in explanations:

| Primary label | Short explanation | Technical detail shown secondarily |
| --- | --- | --- |
| Language tools | Errors, navigation, actions, and suggestions for selected languages | LSP, `nvim-lspconfig`, selected servers |
| External language program | A separate program that reads the project and answers Neovim | language server executable |
| Search and navigation | Search files, text, help, commands, symbols, and open buffers | Telescope and optional native sorter |
| File explorer | Browse folders and reveal the current file | built-in commands or Neo-tree |
| Tool installer | Download selected language servers, formatters, linters, and debuggers | Mason |
| Completion | Suggestions from language tools, paths, and snippets | native completion or `blink.cmp` |
| Plugin | Optional code added to Neovim | managed by `vim.pack` |
| Current session | What the open Neovim process is using now | effective runtime value |
| Saved configuration | Lua that will be loaded on startup | persisted configured value |

Do not use “client,” “server,” “runtime path,” “completion source,” or “executable” as unexplained primary labels.

## 2. Information architecture

### First-run hierarchy

```text
Welcome
`- Start setup
   |- Profile
   |  |- Kickstart
   |  |- General editing
   |  |- Web development
   |  |- Lua development
   |  `- Custom
   |- Editor essentials
   |  |- Display and movement
   |  |- Indentation
   |  |- Editing safety
   |  `- Appearance and diagnostics summary
   |- Development tools
   |  |- Search and navigation
   |  |- Git
   |  |- Languages and LSP
   |  |- Completion and snippets
   |  |- Formatting and linting
   |  `- Treesitter
   |- Optional features
   |  |- File explorer
   |  |- Autopairs and indentation guides
   |  |- Debugging
   |  `- Advanced keymaps
   `- Review
      |- Selected and implied choices
      |- Dependencies
      |- Plugin and external-tool actions
      |- Exact file plan
      |- Generated Lua
      |- Diff
      `- Save and result
```

Welcome and Review are not counted as configuration decision screens. Development tools and Optional features always remain visible, even when a profile selects none; their summaries say exactly what will not be configured and provide a Customize action. This prevents silent skipping without turning onboarding into the full settings catalog.

### Profile decision type

Profile is a **single-choice starting proposal**, followed by progressive optional decisions. It is not a permanent mode and not a multi-select checklist.

- Kickstart reproduces the audited upstream defaults while keeping upstream optional examples off.
- General editing selects a restrained native editing setup without language-specific tools.
- Web development proposes JavaScript/TypeScript, HTML, CSS, JSON, completion, formatting choices, and relevant parsers.
- Lua development proposes `lua_ls`, Stylua, completion, snippets, and relevant parsers.
- Custom starts from native editor behavior and no optional plugins.
- Every profile proposes values; it does not lock them.
- Later screens allow every proposal to be changed independently.
- After onboarding, the profile is retained only as provenance in Overview. Changing profiles later must preview affected values and preserve manual changes by default.

### Baseline editor settings

These ten choices remain the compact editor-essential slice used by the existing wireframes and first vertical implementation. They are not the complete editor catalog:

| Group | Setting | Values | Recommended proposal |
| --- | --- | --- | --- |
| Display | Line number style | Absolute; Relative with absolute current line; Off | Absolute |
| Display | Wrap long lines | On; Off | General: On; code recipes: Off |
| Display | Highlight current line | On; Off | On |
| Indentation | Indentation width | 2; 4; 8 | JS/Lua: 2; General: 4 |
| Indentation | Indent with | Spaces; Tabs | Spaces |
| Editing | Persistent undo | On; Off | On |
| Editing | System clipboard | Use system clipboard; Neovim only | Use system clipboard when available |
| Editing | Confirm before abandoning changes | On; Off | On |
| Search | Search case | Smart; Exact | Smart |
| Appearance | Built-in colors | Neovim default; habamax | habamax |

“Relative with absolute current line” is one enum value rather than two beginner-facing toggles. Its details view explains that it writes both the absolute and relative number options. This prevents invalid or puzzling combinations while still exposing the generated Lua.

In the first editor-only vertical slice, “one built-in colorscheme choice” means one explicit alternative, `habamax`, plus “Neovim default.” The full Appearance catalog adds the audited Tokyo Night variants and validated theme selection.

“Confirm before abandoning changes” controls Neovim's own confirmation behavior for unsaved editing buffers. The configurator's draft-exit guard is always active and does not depend on this setting.

If “Use system clipboard” is selected but Neovim cannot find a usable OS clipboard provider, treat clipboard integration as a pending feature. Saving remains allowed, but Review and Health must say that system copy/paste is not yet available.

The following legacy recipe table defines only the first editor/JavaScript vertical slice. The complete profiles are defined in the configuration catalog. In both cases, proposals are deterministic and availability changes status, not the user's selected intent:

| Choice | General | JavaScript / TypeScript | Lua experimental |
| --- | --- | --- | --- |
| Line number style | Absolute | Absolute | Absolute |
| Wrap long lines | On | Off | Off |
| Highlight current line | On | On | On |
| Indentation width | 4 | 2 | 2 |
| Indent with | Spaces | Spaces | Spaces |
| Persistent undo | On | On | On |
| System clipboard | Use system clipboard | Use system clipboard | Use system clipboard |
| Confirm before abandoning | On | On | On |
| Search case | Smart | Smart | Smart |
| Built-in colors | habamax | habamax | habamax |
| Language tools | None | JavaScript / TypeScript | None |
| File picker proposal | mini.pick | mini.pick | mini.pick |

The code-generation contract for editor choices is also fixed at the product level:

| Setting | Generated behavior |
| --- | --- |
| Line number style | Set Neovim's absolute and relative number options as one coherent pair. Relative keeps an absolute current line. |
| Wrap long lines | Set the line-wrap option. |
| Highlight current line | Set the cursor-line option. |
| Indentation width | Set the tab display width, shift width, and soft-tab width consistently to the selected number. |
| Indent with | Enable or disable expansion of typed tabs into spaces. |
| Persistent undo | Enable or disable Neovim's persistent undo-file behavior; future undo files are runtime data, not files written by the save transaction. |
| System clipboard | Add the system clipboard integration when selected; use Neovim's internal registers when not selected. |
| Confirm before abandoning changes | Set Neovim's confirmation option. |
| Search case | Smart enables case-insensitive search unless the query contains capitals; Exact disables both ignore-case and smart-case behavior. |
| Built-in colors | `habamax` writes one built-in colorscheme selection; Neovim default writes no colorscheme selection. |

The explanation and Generated Lua views show the exact Lua resulting from these behaviors. When one UI decision writes multiple options, the source annotation groups those lines under the single setting.

### Later settings hierarchy

```text
Overview                         <- default landing screen
|- Status: saved, draft, current session, restart
|- Active profile and feature summary
|- Quick actions: edit, review changes, health
`- Independence note

Profiles
Editor
Appearance
Diagnostics
Keymaps
Search
File explorer
Git
Languages and LSP
Completion and snippets
Formatting
Linting
Treesitter
Debugging
Plugins

Health
|- Neovim version
|- Dependencies
|- Plugin availability
|- Generated-file integrity
`- Recheck

Recovery                         <- visible when history or an issue exists
|- Incomplete transaction
|- Last known-good backup
|- Manual-change conflict
`- Restore and verify

Contextual utilities
|- Review changes                <- enabled only when draft is dirty
|- Generated Lua
|- Diff
|- Key help
`- About removal
```

### Sections to combine or omit

- Keep **Appearance**, **Diagnostics**, **Keymaps**, **Search**, **Git**, **Languages and LSP**, **Completion and snippets**, **Formatting**, **Linting**, **Treesitter**, and **Debugging** as independent later-settings sections because the Kickstart-compatible catalog gives each meaningful content.
- Combine dependency status into the feature being edited during onboarding and aggregate it in Health later.
- Keep **Plugins** as a read-oriented inventory of selected plugin requests, versions, dependencies, lockfile state, and availability. It is not a generic marketplace.
- Keep **Search** and **File explorer** separate: a picker and a directory tree solve different problems and may both be selected.
- Do not create a separate Overview during onboarding; Welcome and Review already supply orientation.
- Generated Lua and Diff are views of the draft, not top-level beginner decisions.
- Recovery is normally quiet and secondary. It becomes prominent when a journal, hash mismatch, or invalid generated file is detected.
- Do not expose internal modules, manifest schema, or transaction phases in the primary navigation.

### Command behavior

- `:NvimConfig` is always the documented command.
- `:Settings` may be created only when no command with that exact name already exists and the user has not disabled aliases.
- The alias is never allowed to replace or shadow an existing command.
- Re-running either command focuses the existing configurator tab instead of creating a duplicate.
- After successful onboarding, both commands land on Overview.
- If an incomplete transaction or corrupt state exists, they land on Recovery instead.
- If a non-empty unmanaged configuration exists, they land on the read-only Existing configuration screen.

## 3. End-to-end flows

The editor and JavaScript examples below are the first implementation slice, not the limit of the configuration catalog. The same state, dependency, review, save, failure, and recovery rules apply to every catalog feature.

### 3.1 Fresh first launch

1. The plugin auto-opens only when its first-run marker is present and the target configuration is empty or is a recognized configurator-owned configuration.
2. Welcome states three promises: ordinary Lua, preview before write, and continued operation without the configurator.
3. No filesystem change or dependency installation happens on Welcome.
4. `Enter` on “Start setup” opens Profile.
5. Closing Welcome before a draft exists closes without a warning.
6. Once a recipe creates a draft, close attempts use the unsaved-exit guard.

If startup dimensions are below the supported minimum, show the resize screen before Welcome. If Neovim is older than 0.12, show a blocking compatibility error and do not allow setup or saving.

### 3.2 Choose a profile

1. Kickstart is initially focused. Nothing is selected until the user presses `Enter` or `Space`; focus is not selection.
2. Selecting Kickstart, General editing, Web development, Lua development, or Custom creates a deterministic proposal and shows counts for native settings, plugins, language tools, parsers, keymaps, and external actions.
3. The summary distinguishes Kickstart defaults from Kickstart optional examples. Optional examples remain off unless explicitly selected.
4. The user may continue only after a profile is selected.
5. Returning to this step and changing the profile shows affected values before applying it. Values manually changed after selecting a profile are preserved by default; “Replace my edits with profile values” is a secondary explicit action.

This is single-select because a profile is only a starting proposal. Feature selection becomes progressive and independently editable on later screens.

### 3.3 Configure editor behavior

1. Rows show the draft value, a recommended marker when applicable, and `*` only when the value differs from the persisted configuration.
2. `j`/`k` or arrow keys move between settings; `h`/`l` or left/right cycles enumerated values; `Space` toggles two-state values.
3. The selected row's short description and immediate Lua consequence appear in the details region. On narrower layouts they appear below the row.
4. `?` or `K` opens a focused explanation containing “What it does,” “Recommendation,” and “Generated Lua.”
5. No draft change is applied to the current session.
6. `u` undoes the latest draft mutation. `U` requests a reset to the persisted values, or to the selected recipe when no persisted configuration exists.

### 3.4 Configure language support

The screen first describes the outcome, then the selected parts in data-flow order:

1. **Neovim language support:** built into Neovim; it sends questions and displays answers.
2. **Setup defaults:** `nvim-lspconfig` supplies known configurations for selected servers.
3. **Tool installation:** Mason may acquire explicitly reviewed servers, formatters, linters, and debuggers.
4. **External language program:** the selected server understands the project and answers Neovim.
5. **Completion:** native completion or `blink.cmp`, with all resulting keymaps disclosed.
6. **Related tools:** selected parser, formatter, linter, snippets, and debugger recipe.

Language recipes are independently selectable. The first implemented recipes are Lua and JavaScript/TypeScript; the catalog also covers the Kickstart examples for C/C++, Go, Python, and Rust. Selecting a language shows every implied plugin, tool, parser, mapping, and download in details and Review. Removing a language removes those implications only when no other selected feature requires them.

The Completion explanation spells out the selected engine's actual keys. It must show whether `Ctrl-Space`, `Ctrl-N`, `Ctrl-P`, `Ctrl-Y`, `Ctrl-E`, `Ctrl-K`, Tab, or Enter are mapped, and warn before Enter is changed from newline to completion acceptance.

### 3.5 Handle missing dependencies

1. Checks run when Language tools opens, when Review opens, immediately before save, and when the user presses `r`.
2. Every dependency row has status, purpose, impact, and last-check time.
3. Availability uses `[FOUND]`, `[MISSING]`, `[CHECKING]`, and `[UNKNOWN]`. Requirement is a separate label: `[REQUIRED FOR ...]` or `[OPTIONAL]`.
4. Selecting a missing row opens OS-appropriate installation guidance. The configurator does not run package-manager or global `npm` commands automatically.
5. “Recheck” reruns detection and updates the existing row; it never assumes installation succeeded.

Dependency policy:

| Dependency | Needed for | Effect when missing |
| --- | --- | --- |
| Neovim 0.12+ | Entire product | Blocks the configurator and all saves |
| `git` | Plugins managed through `vim.pack` | Blocks plugin acquisition; does not roll back verified local configuration |
| `make`, compiler, or plugin-specific build tool | Conditional plugin build | Blocks only that build or its accelerated feature |
| `rg` and `fd` | Fast search and file discovery | Degrades or blocks only the affected search action |
| Tree-sitter CLI/build prerequisites | Parser acquisition | Leaves selected parser features pending |
| Selected runtime/package manager | Language tool acquisition or execution | Blocks only dependent language features |
| Selected server, formatter, linter, or debugger | Its configured feature | Leaves only that feature pending |
| System clipboard provider | Copying between Neovim and other applications | Blocks only system clipboard integration |

When a selected feature dependency is missing, Review says “Save is allowed; these features will remain pending” and lists them. The save confirmation requires acknowledgment of that scoped outcome. If a dependency disappears between preflight and the external phase, the result names the unavailable feature rather than reporting a generic save failure.

### 3.6 Review configuration

Review is the only route to the final save confirmation. It shows:

- selected recipe and explicit settings;
- implied settings, each with “because …” provenance;
- requested plugins and whether each is required or optional;
- missing dependencies and exactly what they block;
- exact absolute paths grouped as Create, Modify, External download, and Configurator record;
- whether any current file will be backed up;
- restart requirement;
- links to Generated Lua and Diff.

Collapsed summary groups show counts, but warnings, missing dependencies, changed paths, and restart state are never collapsed by default.

### 3.7 Inspect generated Lua

1. `p` opens a read-only code view from any screen after a recipe exists.
2. The preview is generated from the current draft, not from the current disk state.
3. Wide terminals show a file list and code side by side; smaller terminals show one file with a file chooser.
4. Code never soft-wraps. Horizontal scrolling remains available.
5. Annotation lines such as `From: Editor > Line number style` are real buffer text in an optional annotation mode or are duplicated in accessible text; extmark-only text is not the sole explanation.
6. `Enter` on an annotation opens the related setting. `b` returns to the previous screen and restores its focused item.
7. Syntax errors link to the preview line and source setting.

### 3.8 Preview diff

1. `d` creates a read-only comparison between the last persisted configurator-owned files and the proposed generated files.
2. At 120 columns or wider, use native side-by-side diff windows.
3. Below 120 columns, use a single unified diff buffer. Two very narrow code panes are less native in practice because constant horizontal scrolling obscures the comparison.
4. New files compare against an explicit `[file does not exist]` state.
5. No diff buffer is modifiable, and `do`, `dp`, writes, and other merge actions are disabled.
6. `[c` and `]c` move between changes. `Enter` on a file heading opens that generated file. `s` returns to Review and opens the save confirmation; it does not write directly from the diff.

### 3.9 Save successfully

The progress screen is divided into two phases.

**Phase A: Local configuration transaction**

1. Validating
2. Backing up
3. Writing
4. Verifying

Only after Verifying succeeds may the UI show `[SAVED]` for local configuration.

**Phase B: External plugin acquisition**

5. Installing requested plugins
6. Verifying plugin availability

The first save may show Backing up as `[SKIPPED] First generated configuration`. A later save records the backup path. When both phases succeed, the result says “Saved and verified” and “Plugins installed” separately. It also says the current session was not changed and a restart is required.

### 3.10 Save and validation failures

All failures preserve the draft and report whether any disk content changed.

- **Generated Lua validation:** fail before writing; show file, line, plain-language reason, and related setting. Previous files are untouched.
- **File-write failure:** stop the transaction, restore the last known-good files when any target was replaced, verify the restoration, and report the result. If restoration cannot be proven, enter Recovery and mark the state incomplete.
- **Missing executable:** treat as a scoped preflight or external-feature failure. State “No files changed” if preflight stopped; if the local phase already completed, state “Configuration saved” and identify only the unavailable feature.
- **Plugin download failure:** never roll back a verified editor configuration. Mark each plugin `[FAILED]`, keep the generated plugin request, and offer Retry, View details, or return to the owning feature and disable or replace it.
- **Manually modified generated file:** stop before backup/write. Show the exact path and three actions: View disk changes; Cancel and keep the manual file; Back up the manual file and replace it after explicit confirmation. Do not offer automatic merge in v0.1.
- **Incomplete previous transaction:** route directly to Recovery on open. Show which phase was last recorded and whether each target matches the old, temporary, or proposed hash. Recommend Restore last known-good unless every final file can be verified as the complete proposed set.

### 3.11 Reopen settings later

1. `:NvimConfig` or the guarded `:Settings` alias opens Overview.
2. Overview shows `Saved`, `Current session matches` or `[RESTART]`, dependency health, and `No draft changes`.
3. The user opens Editor, changes one value, and sees one dirty row.
4. `d` shows a one-setting, usually one-line diff.
5. `s` routes to Review; a confirmation starts the same safe transaction.
6. Success states that the saved value will be used after restart. The UI does not claim it was applied to the current session.

### 3.12 Exit with unsaved changes

The main scratch buffer's modified flag mirrors draft dirtiness so native `:q`, `:tabclose`, and `:qa` also receive Neovim's modified-buffer protection. Buffer-local `q`, `ZZ`, and UI close actions route through the configurator guard.

The confirmation choices are:

1. Keep editing — default and initially focused.
2. Review and save.
3. Discard draft and close.

`Esc`, `Ctrl-C`, or `q` inside the dialog means Keep editing. “Discard” includes the number of draft changes. `:q!` and `ZQ` are still treated as explicit discard intentions but receive one configurator confirmation because a scratch buffer otherwise makes their consequence unclear.

### 3.13 Existing configuration detected

Classify a configuration as unsupported when the target config root is non-empty and lacks a valid configurator ownership manifest. A recognized generated configuration with matching hashes is supported; a recognized one with mismatched hashes is a manual-change conflict.

For an unsupported existing configuration:

- show “Read-only — nothing was modified” at the top;
- list the detected entry files and absolute paths;
- explain that safe merging requires understanding arbitrary user code and is outside v0.1;
- allow read-only exploration of recipes, settings, generated Lua, and a hypothetical diff labelled `PREVIEW ONLY`;
- disable Save everywhere and explain why when pressed;
- offer Open existing file, Explore preview, or Close;
- never offer Delete, Rename, Move, or “Take over this config.”

### 3.14 Recovery and rollback

Recovery is state-driven, not a generic backup browser.

- **Healthy history:** show the current generated version and the most recent last known-good backup.
- **Incomplete transaction:** show the journal phase and per-file evidence; recommend the safest valid action.
- **Corrupt manifest:** disable edits. If a verified backup exists, offer restoration. Without a verified ownership record or backup, fall back to read-only existing-config behavior rather than adopting files automatically.
- **Manual changes:** preserve them by default and offer a comparison. Replacement always creates a new backup of the manual version first.
- **Rollback:** Preview backup, then Restore. Restoration is itself journaled, verified, and followed by a restart notice. The current files are backed up before restoration, so rollback can be undone.

After recovery, Health must show whether generated Lua parses, hashes match, requested plugins are available, and a restart is needed. Never mark recovery complete until verification succeeds.

### 3.15 Removing the configurator

Overview and Health include “What happens if I remove the configurator?” The explanation separates what remains from what disappears:

- **Remains:** `init.lua`, generated `lua/user_config/` files, built-in plugin lockfile, installed plugins, and normal Neovim startup behavior.
- **Disappears:** `:NvimConfig`, the optional `:Settings` alias, guided editing, UI validation, and UI recovery commands.
- **Not done automatically:** deleting generated configuration, uninstalling requested plugins, or deleting backups.

The configurator provides explanation, not an in-product uninstall button. Removal belongs to however the user installed the configurator, and the UI must show those instructions without implying that generated configuration depends on it.

## 4. ASCII wireframes

These wireframes preserve the first editor/JavaScript/file-picker vertical slice and the interaction language it established. The complete product adds the catalog sections without changing the focus, selection, state, accessibility, review, diff, save, and recovery conventions shown here.

### Wireframe conventions

These drawings use ASCII deliberately. The production renderer may use the same characters everywhere; Unicode decoration is unnecessary.

```text
>            focused row
(x) / ( )    selected / unselected single choice
[x] / [ ]    enabled / disabled toggle
*            draft differs from saved configuration
[OK]         completed, found, or verified
[MISSING]    not detected
[PENDING]    configured but not yet available
[RESTART]    saved but not active in this session
[ERROR]      action failed
```

The outer boxes below communicate terminal bounds. The actual full-screen interface should use whitespace and one or two separators rather than boxing every region.

### 4.1 Welcome — normal, 80 columns

```text
+------------------------------------------------------------------------------+
| nvim-configurator                                                           |
| Build your Neovim from inside Neovim.                                        |
|------------------------------------------------------------------------------|
|                                                                              |
|  Create a useful Neovim configuration and learn what each choice writes.     |
|                                                                              |
|  - You will review ordinary Lua before anything is saved.                    |
|  - You will see every file that may change.                                  |
|  - Your configuration keeps working if this configurator is removed.         |
|                                                                              |
|  This setup is for an empty Neovim configuration.                            |
|                                                                              |
|  > Start setup                                                               |
|    What will be configured?                                                  |
|                                                                              |
|------------------------------------------------------------------------------|
| Enter Open   j/k Move   g? All keys   q Close                                |
+------------------------------------------------------------------------------+
```

“What will be configured?” opens a short explanation float; it does not add another onboarding step.

### 4.2 Welcome — narrow, 52 columns

```text
+--------------------------------------------------+
| nvim-configurator                               |
| Build your Neovim from inside Neovim.            |
|--------------------------------------------------|
|                                                  |
| Create a useful configuration and learn what     |
| every choice writes.                             |
|                                                  |
| - Review Lua before saving                       |
| - See every changed file                         |
| - Keep the config without this plugin            |
|                                                  |
| > Start setup                                    |
|   What will be configured?                       |
|                                                  |
|--------------------------------------------------|
| Enter Open  j/k Move  g? Keys  q Close           |
+--------------------------------------------------+
```

### 4.3 Intended use — normal

```text
+------------------------------------------------------------------------------+
| nvim-configurator > Intended use (1 of 5)          Draft: not yet selected   |
|------------------------------------------------------------------------------|
| What will you edit first?                                                   |
| This only chooses starting values. You can change every choice next.         |
|                                                                              |
| > ( ) General editing                     [RECOMMENDED IF UNSURE]             |
|       Comfortable defaults for notes, text, and mixed files.                 |
|                                                                              |
|   ( ) JavaScript / TypeScript                                                |
|       Code-oriented indentation plus optional language tools.                |
|                                                                              |
|   ( ) Lua                                      [EXPERIMENTAL RECIPE]          |
|       Code-oriented defaults. No Lua language server in v0.1.                |
|                                                                              |
|   Continue                                                                   |
|                                                                              |
|------------------------------------------------------------------------------|
| Space/Enter Select   j/k Move   ? Explain   b Back   q Close                 |
+------------------------------------------------------------------------------+
```

Moving focus does not change `( )` to `(x)`. After selection, the header changes to `Draft: General editing (not saved)` and Continue becomes active.

### 4.4 Editor settings — wide, 120+ columns

```text
+----------------------------------------------------------------------------------------------------------------------+
| nvim-configurator   Editor (2 of 5)                                  Draft: 2 changes | Current session unchanged    |
|----------------------------------------------------------------------------------------------------------------------|
| Steps                | Editor behavior                              | About selected                               |
|                      |                                              |                                              |
| [OK] Intended use    | Display                                      | Line number style                            |
|  >   Editor          | >  Line number style       Absolute          | Shows a stable number beside each line.      |
|      Language tools  |    Wrap long lines         Off               |                                              |
|      File picker     |    Highlight current line  On                | Recommended: Absolute for new users.         |
|      Review          |                                              |                                              |
|                      | Indentation                                  | Draft: Absolute                              |
|                      |    Indentation width       2          *       | Saved: Relative                              |
|                      |    Indent with            Spaces             | Current session: Relative                    |
|                      |                                              | Neovim default: Absolute                     |
|                      | Editing and search                           |                                              |
|                      |    Persistent undo        On                 | Generated Lua: 2 lines                       |
|                      |    System clipboard       Use system         | ? opens the complete explanation.            |
|                      |    Confirm before abandoning On              |                                              |
|                      |    Search case            Smart      *       |                                              |
|                      |                                              |                                              |
|                      | Built-in appearance                         |                                              |
|                      |    Colors                  habamax            |                                              |
|                      |                                              |                                              |
|                      |    Continue                                  |                                              |
|----------------------------------------------------------------------------------------------------------------------|
| j/k Move  h/l Change  Space Toggle  ? Explain  p Lua  d Diff  u Undo  U Reset  s Review  q Close                    |
+----------------------------------------------------------------------------------------------------------------------+
```

The right region is contextual, not a persistent editable form. All state words that matter also exist as real buffer text, not only extmarks.

### 4.5 Editor settings — normal, 80–119 columns

```text
+------------------------------------------------------------------------------+
| nvim-configurator > Editor (2 of 5)       Draft: 2 changes | not applied now |
|------------------------------------------------------------------------------|
| Display                                                                      |
| > Line number style                                  Absolute                |
|   Shows a stable number beside each line.                                     |
|   Wrap long lines                                    Off                     |
|   Highlight current line                             On                      |
|                                                                              |
| Indentation                                                                  |
|   Indentation width                                   2              *       |
|   Indent with                                        Spaces                  |
|                                                                              |
| Editing and search                                                         |
|   Persistent undo                                    On                      |
|   System clipboard                                   Use system              |
|   Confirm before abandoning changes                  On                      |
|   Search case                                        Smart          *       |
|                                                                              |
| Built-in appearance                                                         |
|   Colors                                              habamax                 |
|                                                                              |
|   Continue                                                                   |
|------------------------------------------------------------------------------|
| j/k Move  h/l Change  ? Explain  p Lua  d Diff  u Undo  s Review  g? Keys    |
+------------------------------------------------------------------------------+
```

Only the focused row's one-line description is inserted below it. This gives context without doubling the height of every setting.

### 4.6 Editor settings — narrow, below 80 columns

```text
+--------------------------------------------------+
| Editor (2 of 5)                                  |
| Draft: 2 changes | not applied now               |
|--------------------------------------------------|
| Display                                          |
| > Line number style                              |
|     Absolute                                     |
|   Shows a stable number beside each line.        |
|                                                  |
|   Wrap long lines                                |
|     Off                                          |
|   Highlight current line                         |
|     On                                           |
|                                                  |
| Indentation                                      |
|   Indentation width *                            |
|     2                                            |
|   Indent with                                    |
|     Spaces                                       |
|                         [more below]             |
|--------------------------------------------------|
| j/k Move  h/l Change  ? Explain                  |
| p Lua  d Diff  s Review  g? All keys             |
+--------------------------------------------------+
```

Values move to their own indented line. The UI scrolls the focused row into view and reserves the final two lines for actions.

### 4.7 Setting explanation — normal/wide float

```text
             +----------------------------------------------------------+
             | Line number style                                   [x] |
             |----------------------------------------------------------|
             | What it does                                             |
             | Shows where you are in a file. Relative numbers show     |
             | the distance to nearby lines for motion commands.        |
             |                                                          |
             | Recommendation                                           |
             | Absolute is easier while learning.                       |
             |                                                          |
             | Your values                                              |
             | Neovim default: Absolute   Saved: Relative               |
             | Draft: Absolute          Current session: Relative       |
             |                                                          |
             | Generated Lua                                            |
             | Two related Neovim options will be written.              |
             |                                                          |
             | Enter View Lua   Esc/q Close                             |
             +----------------------------------------------------------+
```

The float is at most 64 columns and at most 70% of available height. `[x]` is a literal close label, not an icon-only control. In a narrow terminal the explanation becomes a centered float with two-column value pairs stacked vertically; if it cannot fit above the minimum size, it opens as a temporary full-screen detail buffer instead.

### 4.8 Setting explanation — narrow

```text
  +--------------------------------------------+
  | Line number style                    [x]  |
  |--------------------------------------------|
  | What it does                              |
  | Shows where you are in a file.            |
  |                                           |
  | Recommendation                            |
  | Absolute is easier while learning.        |
  |                                           |
  | Default: Absolute                         |
  | Saved: Relative                           |
  | Draft: Absolute                           |
  | Current session: Relative                 |
  |                                           |
  | Enter View Lua   Esc/q Close              |
  +--------------------------------------------+
```

### 4.9 Language setup — normal

```text
+------------------------------------------------------------------------------+
| nvim-configurator > Language tools (3 of 5)      Selected: JavaScript / TS   |
|------------------------------------------------------------------------------|
| > (x) JavaScript / TypeScript tools                                           |
|       Errors, navigation, and completion suggestions for JS and TS files.    |
|   ( ) None                                                                    |
|       Do not configure language-specific tools.                               |
|                                                                              |
| How the parts work together                                                   |
|                                                                              |
| > [BUILT IN] Neovim language support                                         |
|     Sends requests and shows answers inside the editor.                      |
|                                                                              |
|   [WILL ADD] nvim-lspconfig                                                   |
|     Supplies known setup defaults. It is not the language program.           |
|                                                                              |
|   [SETUP NAME] ts_ls                                                          |
|     The name Neovim uses for this JavaScript/TypeScript setup.                |
|                                                                              |
|   [MISSING] typescript-language-server                                        |
|     External program that understands your project. JS tools cannot start.   |
|                                                                              |
|   [BUILT IN] Completion                                                       |
|     Neovim 0.12 shows suggestions; no completion plugin will be added.        |
|                                                                              |
|   Continue                                                                    |
|------------------------------------------------------------------------------|
| Space/Enter Select  j/k Move  r Recheck  ? Explain  b Back  q Close          |
+------------------------------------------------------------------------------+
```

When None is selected, the parts list remains visible but disabled and starts with “These parts will not be configured.” This teaches without implying selection.

### 4.10 Missing dependencies — normal

```text
+------------------------------------------------------------------------------+
| Dependencies                                         Checked just now        |
|------------------------------------------------------------------------------|
| Program                         Status       Needed for                       |
| > node                          [FOUND]      JavaScript language program      |
|   typescript-language-server    [MISSING]    JavaScript language tools        |
|   git                           [FOUND]      Downloading requested plugins    |
|   rg                            [MISSING]    Optional: faster picker search   |
|                                                                              |
| typescript-language-server is missing.                                       |
|                                                                              |
| What this affects: JavaScript errors, navigation, and suggestions.           |
| What still works: saving editor settings and all ordinary editing.           |
|                                                                              |
| Installation guidance                                                        |
| 1. Install it using the package command shown for your system.               |
| 2. Return here and press r.                                                   |
| 3. Wait for [FOUND]. The configurator does not assume success.               |
|                                                                              |
| > Show command for this system                                                |
|   Recheck now                                                                 |
|------------------------------------------------------------------------------|
| j/k Move  Enter Open  r Recheck  y Copy shown command  b Back  g? All keys   |
+------------------------------------------------------------------------------+
```

The copy action is active only when a concrete command is displayed. It copies text; it never executes it. If clipboard support is unavailable, the row says “Copy unavailable — select the command text.”

### 4.11 Missing dependencies — narrow

```text
+--------------------------------------------------+
| Dependencies                       checked now   |
|--------------------------------------------------|
| > node                                           |
|     [FOUND]                                      |
|     Needed for: JS language program              |
|                                                  |
|   typescript-language-server                     |
|     [MISSING]                                    |
|     Blocks: JS language tools                    |
|     Does not block: editor settings              |
|                                                  |
|   git                                            |
|     [FOUND]                                      |
|                                                  |
|   rg                                             |
|     [MISSING] [OPTIONAL]                         |
|--------------------------------------------------|
| Enter Details  r Recheck  b Back  g? Keys        |
+--------------------------------------------------+
```

### 4.12 Optional file picker — normal

```text
+------------------------------------------------------------------------------+
| nvim-configurator > File picker (4 of 5)                                     |
|------------------------------------------------------------------------------|
| How would you like to find and open files?                                   |
|                                                                              |
| > (x) mini.pick                                      [RECOMMENDED]            |
|       A small optional plugin with a searchable file list.                   |
|       Will request: mini.pick                                                 |
|       Needs: git [FOUND]   rg [OPTIONAL, MISSING]                             |
|                                                                              |
|   ( ) None                                                                    |
|       Keep Neovim's built-in file-opening commands only.                     |
|       No file-picker plugin will be requested.                               |
|                                                                              |
|   Continue                                                                    |
|                                                                              |
| mini.pick is optional. Choosing None does not reduce language support.        |
|------------------------------------------------------------------------------|
| Space/Enter Select  j/k Move  ? Explain  b Back  p Lua  s Review  q Close    |
+------------------------------------------------------------------------------+
```

In this legacy picker slice there is no generic “Plugins” marketplace and no “Other” choice. The complete product adds a read-oriented Plugins inventory for selected catalog features, versions, dependencies, lockfile state, and availability.

### 4.13 Review summary — wide

```text
+----------------------------------------------------------------------------------------------------------------------+
| nvim-configurator   Review (5 of 5)                              Ready to write | 1 feature pending                  |
|----------------------------------------------------------------------------------------------------------------------|
| Steps                | Your configuration                           | Save readiness                              |
|                      |                                              |                                             |
| [OK] Intended use    | Recipe                                       | [OK] Generated Lua is valid                 |
| [OK] Editor          |   JavaScript / TypeScript                    | [OK] Target is writable                     |
| [!] Language tools   |                                              | [!] JS tools are pending                    |
| [OK] File picker     | Editor                          10 settings  |     typescript-language-server missing      |
|  >   Review          |   8 recipe values, 2 changed                  |     Does not block config write             |
|                      |                                              |                                             |
|                      | Implied                                      | Activation                                  |
|                      |   nvim-lspconfig (because JS tools selected)  | [RESTART] Required after save               |
|                      |   ts_ls setup (because JS tools selected)     | Current session will not be changed         |
|                      |   native completion                           |                                             |
|                      |                                              | Exact paths                                 |
|                      | Requested plugins                             | CREATE /.../nvim/init.lua                   |
|                      |   nvim-lspconfig             Required         | CREATE /.../lua/user_config/editor.lua      |
|                      |   mini.pick                  Optional         | CREATE /.../lua/user_config/languages.lua   |
|                      |                                              | CREATE /.../lua/user_config/plugins.lua     |
|                      | Missing                                      | MAY CHANGE /.../nvim-pack-lock.json         |
|                      |   typescript-language-server                  | RECORD /.../state/.../manifest.json         |
|                      |   Blocks JS tools only                        |                                             |
|----------------------------------------------------------------------------------------------------------------------|
| Enter Expand  p Generated Lua  d Diff  b Back  s Save  ? Explain warning  q Close                                   |
+----------------------------------------------------------------------------------------------------------------------+
```

Absolute paths must be rendered in the real interface. The abbreviated `/.../` form is only for this document's width.

### 4.14 Review summary — normal/narrow behavior

At 80–119 columns, Save readiness appears first, followed by vertically stacked summary groups. Below 80 columns, each group is a single actionable row with its status and count:

```text
+--------------------------------------------------+
| Review                                           |
| Ready to write | 1 feature pending               |
|--------------------------------------------------|
| > [OK] Validation                               |
|   [!] Missing dependencies             1        |
|       JS tools pending; config can be saved      |
|   [OK] Editor settings                 10        |
|   [OK] Implied choices                  3        |
|   [OK] Requested plugins                2        |
|   [OK] Files to create                  4        |
|   [RESTART] Activation                           |
|                                                  |
| Press Enter to expand the focused group.         |
|--------------------------------------------------|
| p Lua  d Diff  b Back  s Save  ? Explain         |
+--------------------------------------------------+
```

### 4.15 Generated Lua — wide

```text
+----------------------------------------------------------------------------------------------------------------------+
| Generated Lua                                      Draft preview | valid | not written                              |
|----------------------------------------------------------------------------------------------------------------------|
| Files                                | lua/user_config/editor.lua                                                     |
|                                      |                                                                               |
| > init.lua                           |  1  -- Generated configuration: ordinary Lua                                   |
|   lua/user_config/editor.lua         |  2  -- It does not require nvim-configurator.                                 |
|   lua/user_config/languages.lua      |  3                                                                            |
|   lua/user_config/plugins.lua        |  4  -- From: Editor > Line number style                                       |
|                                      |  5  vim.opt.number = true                                                      |
| Filter: all settings                 |  6  vim.opt.relativenumber = false                                             |
| Annotations: shown                   |  7                                                                            |
|                                      |  8  -- From: Editor > Wrap long lines                                         |
|                                      |  9  vim.opt.wrap = false                                                       |
|                                      | 10                                                                            |
|                                      | 11  -- From: Editor > Indentation width                                       |
|                                      | 12  vim.opt.shiftwidth = 2                                                     |
|                                      |                                                                               |
|----------------------------------------------------------------------------------------------------------------------|
| j/k Scroll  Enter Open file/setting  a Annotations  / Search  d Diff  b Back  s Review and save  q Back             |
+----------------------------------------------------------------------------------------------------------------------+
```

The actual comments and generated lines are part of the preview content. Supplementary provenance may use extmarks, but it may not be the only place a critical fact appears.

### 4.16 Generated Lua — normal and narrow

```text
+------------------------------------------------------------------------------+
| Generated Lua > editor.lua                  Draft preview | valid | unwritten |
|------------------------------------------------------------------------------|
| File 2 of 4   Enter: choose file   Annotations: shown                         |
|                                                                              |
|  1  -- Generated configuration: ordinary Lua                                 |
|  2  -- It does not require nvim-configurator.                                |
|  3                                                                           |
|  4  -- From: Editor > Line number style                                      |
|  5  vim.opt.number = true                                                     |
|  6  vim.opt.relativenumber = false                                            |
|                                                                              |
|                    Code does not wrap; use zh/zl to scroll.                   |
|------------------------------------------------------------------------------|
| j/k Scroll  Enter Files  / Search  d Diff  b Back  g? Keys                   |
+------------------------------------------------------------------------------+
```

Below 80 columns the same single-file layout is used. The file chooser becomes a small full-width list rather than a float if it cannot fit. Code remains unwrapped, and the current horizontal offset is shown as `Column 18` when nonzero.

### 4.17 Diff preview — wide, native side-by-side

```text
+----------------------------------------------------------------------------------------------------------------------+
| Proposed changes                                  4 files | 12 additions | 0 removals                                |
|----------------------------------------------------------------------------------------------------------------------|
| CURRENT: lua/user_config/editor.lua                 | PROPOSED: lua/user_config/editor.lua                           |
|                                                     |                                                                  |
|  1  vim.opt.number = true                           |  1  vim.opt.number = true                                       |
| -2  vim.opt.relativenumber = true                   | +2  vim.opt.relativenumber = false                              |
|  3  vim.opt.wrap = false                            |  3  vim.opt.wrap = false                                        |
|                                                     |                                                                  |
| Native diff mode | change 1 of 2                    | Native diff mode | change 1 of 2                               |
|----------------------------------------------------------------------------------------------------------------------|
| ]c/[c Next/previous change  Tab Switch pane  Ctrl-d/u Scroll  Enter Generated Lua  b Back  s Review  q Back          |
+----------------------------------------------------------------------------------------------------------------------+
```

The two code areas are separate read-only windows with synchronized native diff scrolling. The title/footer lives in a stable controller window or is duplicated without becoming part of the diff.

### 4.18 Diff preview — normal and narrow, unified

```text
+------------------------------------------------------------------------------+
| Proposed changes                                    1 file | 1 setting       |
|------------------------------------------------------------------------------|
| lua/user_config/editor.lua                                                  |
| @@ Line number style                                                         |
| - vim.opt.relativenumber = true                                              |
| + vim.opt.relativenumber = false                                             |
|                                                                              |
| Saved configuration is compared with the current draft.                      |
| This preview is read-only.                                                   |
|------------------------------------------------------------------------------|
| ]c/[c Change  Enter Lua  b Back  s Review and save  q Back                   |
+------------------------------------------------------------------------------+
```

On first run the heading says `NEW FILE` and the current side says `[file does not exist]`. In read-only exploration it says `PREVIEW ONLY - saving is disabled` on every screenful.

### 4.19 Save confirmation — warning acknowledgement

```text
             +----------------------------------------------------------+
             | Save this configuration?                                 |
             |----------------------------------------------------------|
             | Local files                                              |
             |   Create 4 configuration files                           |
             |   Create configurator ownership record                   |
             |                                                          |
             | External phase                                           |
             |   Request 2 plugins                                      |
             |                                                          |
             | [!] JavaScript tools will remain pending because         |
             |     typescript-language-server is missing.               |
             |                                                          |
             | > Save configuration with JS tools pending               |
             |   Back to review                                         |
             |                                                          |
             | Enter Confirm   Esc/q Cancel                              |
             +----------------------------------------------------------+
```

Without warnings the primary action is simply “Save configuration.” The first focused action in a destructive replacement or rollback dialog is always Cancel; ordinary first save may focus Save.

### 4.20 Saving/progress

```text
+------------------------------------------------------------------------------+
| Saving configuration                                                         |
| Do not close Neovim until the local configuration is verified.               |
|------------------------------------------------------------------------------|
| Local configuration                                                          |
|   [DONE] Validating generated Lua                                             |
|   [DONE] Backing up previous configuration                                    |
|          /absolute/path/to/backups/2026-09-03T14-22-10                        |
| > [....] Writing 3 of 4 files                                                 |
|   [WAIT] Verifying written files                                              |
|                                                                              |
| External plugins                                                             |
|   [WAIT] Installing nvim-lspconfig                                            |
|   [WAIT] Installing mini.pick                                                 |
|                                                                              |
| Local configuration is not marked saved until Verifying completes.           |
|------------------------------------------------------------------------------|
| Save in progress. q, Esc, and Ctrl-C do not interrupt the local transaction. |
+------------------------------------------------------------------------------+
```

Use textual state changes rather than animation. A spinner is optional decoration and never replaces `[WAIT]`, `[....]`, `[DONE]`, or `[ERROR]`.

### 4.21 Success

```text
+------------------------------------------------------------------------------+
| Configuration complete                                                       |
|------------------------------------------------------------------------------|
| [SAVED]   Local configuration written and verified                           |
| [OK]      nvim-lspconfig installed                                           |
| [OK]      mini.pick installed                                                 |
| [PENDING] JavaScript language program is still missing                       |
|                                                                              |
| Current session                                                              |
|   Not changed. Restart Neovim to load the saved configuration.               |
|                                                                              |
| Your files are ordinary Lua and do not require nvim-configurator.            |
|                                                                              |
| > Close and restart later                                                     |
|   Open Overview                                                               |
|   View generated Lua                                                          |
|   View saved files                                                            |
|------------------------------------------------------------------------------|
| j/k Move  Enter Open  p Lua  q Close                                          |
+------------------------------------------------------------------------------+
```

If every selected feature is available, the heading remains “Configuration complete.” With any external problem it becomes “Configuration saved — action needed,” not “Success.”

### 4.22 Failure before commit

```text
+------------------------------------------------------------------------------+
| Configuration was not saved                                      [ERROR]     |
|------------------------------------------------------------------------------|
| Writing failed                                                               |
|   /absolute/path/to/nvim/lua/user_config/editor.lua                           |
|                                                                              |
| Reason                                                                       |
|   Neovim could not write the file: permission denied.                        |
|                                                                              |
| Disk result                                                                  |
|   [OK] Previous configuration restored and verified.                         |
|   [OK] Your 2 draft changes are still available in this interface.           |
|                                                                              |
| > Return to Review                                                           |
|   Show technical details                                                     |
|   Open Recovery                                                              |
|                                                                              |
| Nothing is labelled saved.                                                   |
|------------------------------------------------------------------------------|
| j/k Move  Enter Open  y Copy details  q Close (draft guard)                  |
+------------------------------------------------------------------------------+
```

If restoration is unverified, replace the Disk result with `[ERROR] Configuration state is incomplete`, make Open Recovery the primary action, and disable normal saving.

### 4.23 Plugin failure after local save

```text
+------------------------------------------------------------------------------+
| Configuration saved — plugin action needed                                   |
|------------------------------------------------------------------------------|
| [SAVED] Local configuration written and verified                             |
| [OK]    nvim-lspconfig installed                                             |
| [FAILED] mini.pick download                                                   |
|                                                                              |
| Your editor settings are safe. The file picker is not available yet.         |
| The generated plugin setup will not prevent Neovim from starting.            |
|                                                                              |
| > Retry mini.pick download                                                    |
|   Show download details                                                      |
|   Choose None instead (creates a new draft change)                           |
|   Close                                                                       |
|------------------------------------------------------------------------------|
| j/k Move  Enter Open  r Retry  q Close                                        |
+------------------------------------------------------------------------------+
```

### 4.24 Existing configuration detected

```text
+------------------------------------------------------------------------------+
| Existing Neovim configuration detected                    READ-ONLY           |
| Nothing was modified.                                                        |
|------------------------------------------------------------------------------|
| nvim-configurator v0.1 only writes to an empty configuration or to files      |
| it previously generated and can verify. Merging arbitrary Lua safely would   |
| require understanding choices and code that may be unique to you.            |
|                                                                              |
| Detected                                                                     |
|   /absolute/path/to/nvim/init.lua                                             |
|   /absolute/path/to/nvim/lua/options.lua                                      |
|                                                                              |
| > Open existing init.lua                                                     |
|   Explore nvim-configurator as a read-only preview                            |
|   Why is automatic adoption unsafe?                                          |
|   Close                                                                       |
|                                                                              |
| No delete, rename, move, or takeover action is available.                    |
|------------------------------------------------------------------------------|
| j/k Move  Enter Open  ? Explain  q Close                                     |
+------------------------------------------------------------------------------+
```

### 4.25 Recovery

```text
+------------------------------------------------------------------------------+
| Recovery                                                   ACTION REQUIRED   |
|------------------------------------------------------------------------------|
| An earlier save did not finish.                                               |
| Last recorded step: Writing 3 of 4 files                                      |
| Transaction: 2026-09-03 14:22:10                                             |
|                                                                              |
| File status                                                                  |
|   [NEW]  init.lua                           matches proposed version           |
|   [OLD]  lua/user_config/editor.lua         matches last known-good           |
|   [TEMP] lua/user_config/languages.lua      temporary file remains            |
|   [OLD]  lua/user_config/plugins.lua        matches last known-good           |
|                                                                              |
| Recommended                                                                  |
| > Restore the complete last known-good configuration                         |
|   Inspect proposed files                                                      |
|   Show technical details                                                     |
|                                                                              |
| Restoration will first back up the current mixed state, then verify.         |
|------------------------------------------------------------------------------|
| j/k Move  Enter Open  d Compare  ? Explain  q Close without changing files   |
+------------------------------------------------------------------------------+
```

For a simple healthy rollback, replace the incident text with a list of backup timestamps and show Preview before Restore.

### 4.26 Later `:Settings` / `:NvimConfig` landing — wide

```text
+----------------------------------------------------------------------------------------------------------------------+
| nvim-configurator   Overview                                      Saved | Current session matches | No draft changes  |
|----------------------------------------------------------------------------------------------------------------------|
| Sections             | Your Neovim                                  | Details                                     |
|                      |                                              |                                             |
| > Overview           | Configuration                  [SAVED]       | Last saved                                  |
|   Editor             | Current session                [MATCHES]     | 2026-09-03 14:25                            |
|   Language tools     | JavaScript tools               [OK]          |                                             |
|   File picker        | File picker                    mini.pick     | Files                                       |
|   Health             |                                              | 4 generated Lua files                       |
|   Recovery           | Quick actions                                | 2 requested plugins                         |
|                      | > Edit editor behavior                        |                                             |
|                      |   Check dependencies                          | Independence                                |
|                      |   View generated Lua                          | Generated configuration does not require    |
|                      |   View saved files                            | this configurator at startup.               |
|                      |   What happens if I remove the configurator?  |                                             |
|----------------------------------------------------------------------------------------------------------------------|
| j/k Move  Enter Open  / Search settings  p Lua  d Diff  s Review when changed  g? Keys  q Close                    |
+----------------------------------------------------------------------------------------------------------------------+
```

If saved values differ from the current session, the header says `[RESTART] Saved configuration not active` and Details names the affected settings.

### 4.27 Later landing — narrow

```text
+--------------------------------------------------+
| Overview                                         |
| Saved | session matches | no draft               |
|--------------------------------------------------|
| > Configuration                                 |
|     [SAVED] Last saved 14:25                     |
|   Current session                                |
|     [MATCHES]                                    |
|   JavaScript tools                               |
|     [OK]                                         |
|   File picker                                    |
|     mini.pick                                    |
|                                                  |
| Quick actions                                    |
|   Edit editor behavior                           |
|   Check dependencies                             |
|   View generated Lua                             |
|   Removal and independence                       |
|--------------------------------------------------|
| Enter Open  / Search  g? Keys  q Close           |
+--------------------------------------------------+
```

### 4.28 Unsaved-exit confirmation

```text
             +----------------------------------------------------------+
             | Close with 3 unsaved changes?                            |
             |----------------------------------------------------------|
             | Your saved configuration has not changed.                |
             |                                                          |
             | > Keep editing                                           |
             |   Review and save                                        |
             |   Discard 3 draft changes and close                      |
             |                                                          |
             | Enter Choose   Esc/Ctrl-C/q Keep editing                 |
             +----------------------------------------------------------+
```

The selected default is always Keep editing. Discard never happens from a single accidental `q`.

### 4.29 Removal explanation

```text
             +----------------------------------------------------------+
             | If you remove nvim-configurator                         |
             |----------------------------------------------------------|
             | KEEPS WORKING                                            |
             | - Your init.lua and generated Lua                        |
             | - Editor settings and configured language setup          |
             | - Plugins installed separately by Neovim                 |
             |                                                          |
             | NO LONGER AVAILABLE                                      |
             | - :NvimConfig and :Settings                              |
             | - This guided editor and its recovery screens            |
             |                                                          |
             | Nothing will be deleted automatically.                   |
             |                                                          |
             | Enter View files   Esc/q Close                           |
             +----------------------------------------------------------+
```

### 4.30 Below minimum terminal size

The supported minimum is **52 columns by 16 rows**. At smaller sizes, do not attempt a compressed interactive form.

```text
+--------------------------------------+
| nvim-configurator needs more space   |
|                                      |
| Current: 44 columns x 12 rows        |
| Minimum: 52 columns x 16 rows        |
|                                      |
| Resize the terminal. This screen     |
| will update automatically.           |
|                                      |
| No changes have been made.           |
|                                      |
| q Close                              |
+--------------------------------------+
```

No save or selection mapping is active below the minimum.

## 5. Responsive behavior

| Mode | Width | Primary layout | Context and descriptions | Code and diff |
| --- | ---: | --- | --- | --- |
| Wide | 120+ | Later settings: section rail + content + context. Onboarding: steps + content + context. | Persistent context panel for focused row. | File list + code; native side-by-side diff. |
| Normal | 80–119 | Single main content area; onboarding step appears in header. A narrow section rail may be used only when at least 96 columns remain readable. | Focused description appears immediately below its row. | One-file code view and unified diff. |
| Narrow | 52–79 | One column; current section and step in header; no rail. | Values and focused description move below labels. Summary groups become expandable rows. | One-file, unwrapped code view; unified diff only. |
| Unsupported | <52 or <16 rows | Resize message only. | None. | None. |

Height also matters. If fewer than 24 rows are available, contextual explanations should collapse before main settings do. The header may shrink from three lines to two, but the dirty/safety status and footer must remain. Long lists scroll inside the main window; the header and footer stay visible through separate small windows or deterministic redraw regions.

Floats use `min(preferred_width, columns - 4)` and `min(content_height, floor(lines * 0.7))`, with at least two columns and one row of outside space. If content cannot fit, use a normal temporary detail buffer. Modals never extend beyond the current editor grid.

## 6. Component inventory

These are renderer/controller units for this product, not the beginning of a general-purpose UI framework. Prefer small functions operating on explicit screen state over a component class system.

### 6.1 Application screen

- **Responsibility:** Own the dedicated tab, restore the prior tab/window, choose the responsive mode, and compose header, navigation, content, context, and footer windows.
- **State:** route, mode (`first_run`, `settings`, `readonly`, `recovery`), originating window, dimensions, active modal, dirty summary, and transaction lock.
- **Interactions:** route buffer-local keys; handle resize; redirect a repeated `:NvimConfig` call to the existing tab; request guarded close.
- **Rendering:** full tab with a minimal number of normal windows. The main buffer is `nofile`, not listed, read-only outside redraw, and named clearly such as `nvim-configurator://editor`.
- **Accessibility:** critical status is real buffer text. Window titles do not carry unique information. On screen change, put the cursor on the screen heading or first actionable row and announce a concise message through normal Neovim messaging.

### 6.2 Header

- **Responsibility:** Identify the product, route/step, and highest-priority state.
- **State:** title, breadcrumb, step number, saved/dirty state, current-session relation, read-only status, and highest-severity warning.
- **Interactions:** none; the header is not focusable.
- **Rendering:** one to three lines depending on width. Safety labels such as `READ-ONLY`, `[RESTART]`, and `[ERROR]` are never truncated; lower-priority prose truncates first.
- **Accessibility:** state always includes a word or bracketed label, never color alone.

### 6.3 Step/section navigation

- **Responsibility:** Show onboarding progress or later settings sections without exposing internal modules.
- **State:** ordered items, active item, complete/warning/error status, availability, and optional counts.
- **Interactions:** `j`/`k` moves; `Enter` opens. In onboarding, completed earlier steps are open; later steps become open only when prerequisites exist. In later settings all safe sections are open.
- **Rendering:** side rail only when width supports it; otherwise breadcrumb plus a section chooser opened from Overview or `Enter` on the header action.
- **Accessibility:** active item uses `>` and text. Completion uses `[OK]`, not a checkmark alone. Disabled items include a reason in details.

### 6.4 Action list

- **Responsibility:** Provide a common structure for Welcome actions, Overview quick actions, Review groups, errors, and recovery choices.
- **State:** ordered rows with ID, label, optional status, description, enabled state, and activation event.
- **Interactions:** movement, `Enter`, search, and contextual help.
- **Rendering:** one real line for each label and, when needed, one indented description line. Disabled rows remain readable but cannot activate.
- **Accessibility:** the cursor rests on the label line; disabled reason is written adjacent or immediately below.

### 6.5 Setting row

- **Responsibility:** Present one configurable value and connect it to provenance, validation, and generated Lua.
- **State:** ID, label, value type, ordered choices, draft value, persisted value, Neovim default, effective value, recommended value, dirty flag, validity, restart state, and source recipe.
- **Interactions:** `h`/`l` cycles enums, `Space` toggles booleans, `Enter` opens values/details, `?` or `K` explains, and `p` opens code at the related lines.
- **Rendering:** label and draft value are primary. `*`, `[RESTART]`, or `[ERROR]` follows as text. In wide mode all other values live in the context pane; normal/narrow shows only the focused description inline.
- **Accessibility:** value is real text. The dirty marker is explained in header/footer and key help. Extmark highlights may decorate, but never replace, the value or status.

### 6.6 Choice group

- **Responsibility:** Handle recipe and file-picker single-choice decisions.
- **State:** options, selected ID or none, recommended ID, descriptions, implications, and enabled state.
- **Interactions:** `j`/`k`, `Space`/`Enter` select. Selection is explicit; cursor focus alone does not mutate state.
- **Rendering:** `( )` and `(x)` plus labels. Recommendation is a textual suffix. Implications appear below the selected option.
- **Accessibility:** pronounceable ASCII states; no reliance on circle glyphs or color. A selected option reads as `Selected: ...` in its details.

### 6.7 Status badge

- **Responsibility:** Render compact semantic states consistently.
- **State:** kind (`ok`, `missing`, `optional`, `pending`, `checking`, `warning`, `error`, `restart`, `saved`, `readonly`, `disabled`) and optional explanation.
- **Interactions:** not independently focusable unless it represents an actionable warning; then the containing row receives focus.
- **Rendering:** uppercase bracketed word. A highlight group reinforces the meaning.
- **Accessibility:** every state has a distinct word and, where needed, a nearby consequence such as `Blocks: JS tools`.

### 6.8 Context/details panel

- **Responsibility:** Explain the focused setting, dependency, summary group, or action without navigating away.
- **State:** item ID, short description, recommendation, provenance, values, impact, and available related action.
- **Interactions:** normally none; `?` moves the content into an interactive explanation view when links/actions are needed.
- **Rendering:** third column in wide mode; inline block under the focused row in normal mode; omitted until requested in narrow mode except for one essential line.
- **Accessibility:** details are generated as real lines. If the panel is visually separate, the same content is reachable with `?` and announced with its heading.

### 6.9 Footer/action bar

- **Responsibility:** Keep relevant controls discoverable and show transient, non-blocking feedback.
- **State:** current context, active mappings, disabled actions, last action message, and progress lock.
- **Interactions:** none directly.
- **Rendering:** one line wide/normal, two lines narrow. Show at most the six most relevant actions plus `g? All keys`; never horizontally scroll the footer. After an action, a concise message may replace lower-priority hints for several seconds but must also remain in `:messages`.
- **Accessibility:** notation is literal (`Ctrl-C`, `Esc`, `Enter`). Key help explains “hold Ctrl and press C.” Disabled actions are omitted or shown with a reason, never silently accepted.

### 6.10 Explanation modal/detail view

- **Responsibility:** Provide a short definition, recommendation, value comparison, and related Lua or dependency guidance.
- **State:** topic, content blocks, related actions, scroll position, and return focus.
- **Interactions:** `j`/`k` scroll when needed, `Enter` opens the focused related action, `Esc`/`Ctrl-C`/`q` closes.
- **Rendering:** small bordered float when it fits; otherwise a temporary full-screen scratch buffer. It never contains a nested form.
- **Accessibility:** focus moves into the modal, begins at its heading, and returns to the originating row. `[x]` is supplemented by the footer close instruction.

### 6.11 Confirmation dialog

- **Responsibility:** Confirm save, reset, discard, replacement, or rollback with explicit consequences.
- **State:** action type, affected count/paths, choices, safe default, severity, and acknowledgement text.
- **Interactions:** `j`/`k`, arrow keys, `Enter`; `Esc`/`Ctrl-C`/`q` always choose the safe cancel behavior.
- **Rendering:** two or three verb-led choices. Destructive actions include the affected object in their label. Save warnings name unavailable features.
- **Accessibility:** no bare “Yes/No.” The focused action is restated on Enter when the action replaces a manual file or restores a backup.

### 6.12 Search/input view

- **Responsibility:** Find settings, sections, dependencies, and actions by plain-language label or technical alias.
- **State:** query, results, selected result, scope, and return focus.
- **Interactions:** `/` opens; printable keys edit; Backspace removes; `Ctrl-N`/`Ctrl-P` or arrows move results; `Enter` opens; `Esc`/`Ctrl-C` cancels. If `vim.ui.input` is used, results appear after submission rather than requiring a custom insert-mode overlay.
- **Rendering:** a one-line prompt plus a result list. Results show the beginner label first and a breadcrumb second, for example `JavaScript language tools — Language tools`.
- **Accessibility:** matching is not indicated only by highlight. Result count and `No matches` are literal text. Technical searches such as `LSP` and `ts_ls` lead to the beginner-facing item.

### 6.13 Generated-code view

- **Responsibility:** Show the exact Lua produced by the current draft and connect lines back to decisions.
- **State:** generated file set, active file, validation diagnostics, source mapping, annotation visibility, scroll/cursor, and draft/persisted identity.
- **Interactions:** normal read-only movement, file choice, `/` search, `a` annotation toggle, `Enter` on provenance to open a setting, `d` diff, `b`/`q` back.
- **Rendering:** real line numbers, no wrapping, Lua syntax highlight, optional file pane, and a persistent `Draft preview | not written` or `Saved version` label.
- **Accessibility:** essential comments are actual text. Diagnostic text contains file and line. Horizontal offset is stated when nonzero.

### 6.14 Diff view

- **Responsibility:** Compare persisted configurator-owned files with the proposed draft without permitting edits.
- **State:** file diffs, active change, layout mode, source labels, and read-only/exploration flag.
- **Interactions:** native scrolling, `[c`/`]c`, pane switching in wide mode, `Enter` to code, `b`/`q` back, `s` to Review. Mutating diff commands and writes are disabled.
- **Rendering:** native side-by-side diff at sufficient width, otherwise unified diff. Added and removed lines retain `+`/`-` even when highlighted.
- **Accessibility:** changes are encoded by prefix and text, not background color. Every pane has a `CURRENT` or `PROPOSED` heading.

### 6.15 Summary/file-plan group

- **Responsibility:** Show Review facts in expandable groups without hiding risk.
- **State:** category, status, count, rows, expanded state, and severity.
- **Interactions:** `Enter` expands/collapses or opens a detailed path view.
- **Rendering:** wide mode may keep groups expanded; normal/narrow collapses low-risk settings but leaves warnings, path operations, and restart state visible.
- **Accessibility:** category, status, and count are on one line. Expanded/collapsed is written as `[open]` or `[closed]` in linear/narrow representations, not indicated only by a disclosure glyph.

### 6.16 Progress view

- **Responsibility:** Show ordered local and external phases and prevent unsafe interruption.
- **State:** phase, each step's state and timestamps, current path/plugin, committed-local flag, cancellation policy, and error.
- **Interactions:** none during the local transaction except help. After local commit, retry/details actions may appear for external failures. `q`, `Esc`, and `Ctrl-C` do not interrupt a critical write.
- **Rendering:** two labelled phase groups, stable step rows, textual states, backup path, and a sentence stating what is safe. No decorative progress bar is necessary.
- **Accessibility:** each state change is a new line update with a textual word; final results remain visible and are added to `:messages`.

### 6.17 Help view

- **Responsibility:** Explain all mappings, symbols, current state terms, and basic Vim key notation.
- **State:** source context and scroll position.
- **Interactions:** normal scrolling, `/` search, `b`/`q`/`Esc` return.
- **Rendering:** full-screen scratch buffer, not a cramped float. Group mappings by Navigation, Change, Inspect, Save, and Close.
- **Accessibility:** includes literal explanations such as `j: press the j key` and `Ctrl-U: hold Control and press u`; does not assume angle-bracket notation knowledge.

### 6.18 Notification/status message

- **Responsibility:** Confirm reversible actions such as undo, dependency recheck, copied command, or attempted disabled action.
- **State:** message, severity, timestamp, and optional related item.
- **Interactions:** none; errors that require a choice use a screen or modal instead.
- **Rendering:** footer text plus `:messages`; no toast float and no animation.
- **Accessibility:** persists long enough to read and is never the only record of a save or failure.

## 7. Keyboard specification

All mappings are buffer-local to configurator-owned buffers. They must not alter the user's global mappings. The action handler works with stable item IDs rather than cursor line numbers so redraw and responsive changes do not move the logical focus.

| Key | Context | Behavior | Inactive/exception behavior |
| --- | --- | --- | --- |
| `j`, Down | Lists, settings, dialogs | Move to next actionable item; skip headings and prose. | At last item, remain there; do not wrap. In an input field, `j` types normally. |
| `k`, Up | Lists, settings, dialogs | Move to previous actionable item. | At first item, remain there. In an input field, `k` types normally. |
| `gg` | Main lists, code, help | First actionable row or first line. | No effect in confirmation dialogs or input. |
| `G` | Main lists, code, help | Last actionable row or last line. | No effect in confirmation dialogs or input. |
| `Ctrl-D`, `Ctrl-U` | Scrollable main/detail/code/diff/help | Native half-page scroll while preserving the nearest valid focus. | No effect in a short modal or during progress. |
| `h`, Left | Setting row | Select previous enum value. | On non-setting rows, no mutation; footer says `No previous value here`. In code/diff, use native horizontal movement/scroll behavior. |
| `l`, Right | Setting row | Select next enum value. | Same rules as `h`. |
| `Space` | Boolean setting or single-choice option | Toggle a boolean or explicitly select the focused option. | On actions and prose, no effect with a short footer hint. |
| `Enter` | Any actionable row | Open, select, expand, confirm, or continue according to the row's visible verb. | Does not silently change a focused radio option unless the row says Select. |
| `?`, `K` | Main lists/settings/dependencies | Open the focused item's explanation. Both keys are synonyms. | If no item-specific help exists, open screen help. In an input, `?` types and `K` types. |
| `g?` | All non-input configurator views | Open complete key and symbol help. | During a locked save, opens help only after the critical phase finishes; until then prints the essential lock rule. |
| `/` | Main settings and read-only views | Open plain-language search for sections, settings, dependencies, and technical aliases. | In dialogs/progress, unavailable. Code/help may use native buffer search instead. |
| `p` | Any configured draft/saved screen | Open Generated Lua at the focused setting when a source mapping exists. | Disabled on Welcome, resize blocker, corrupt ownership, or before a recipe exists; reason is shown. |
| `d` | Draft/review/generated/recovery screens | Open the appropriate read-only diff. | Disabled when there is no comparable baseline unless first-run new-file preview is available. In unsupported config exploration, opens `PREVIEW ONLY`. |
| `[c`, `]c` | Diff | Move to previous/next change using native diff semantics. | At boundary, stay and report `First/last change`. |
| `Tab`, `Shift-Tab` | Wide code/diff panes; dialogs | Switch pane or move among dialog choices. | On main setting lists, do not change values or route. In text input, Tab is not inserted; it moves to results/action if present. |
| `zh`, `zl` | Code and unified diff | Scroll horizontally left/right. | No effect elsewhere. |
| `a` | Generated Lua | Toggle source annotations. | No effect elsewhere. |
| `r` | Dependencies, Health, external failure | Recheck dependencies or retry the explicitly focused external operation. | Never retries a write or rollback. Elsewhere gives a context hint. |
| `y` | Displayed copyable command/path/details | Copy the explicitly displayed text to a usable clipboard provider. | If clipboard is unavailable, explain how to select the text; never execute it. |
| `u` | Dirty draft screens | Undo the latest configurator draft mutation. | Does not use buffer text undo. With no draft history, says `Nothing to undo`. Disabled during save. |
| `U` | Dirty draft screens | Open confirmation to reset all draft values to persisted values; on first run, to the selected recipe. | Never resets immediately. Disabled during save and read-only mode. |
| `b`, Backspace | Onboarding steps, code, diff, help, detail | Return to the previous logical screen and focused item. | On root Overview/Welcome, remains and says `Already at the first screen`. Backspace edits text inside input. |
| `s` | Draft screens | From edit screens, go to Review. From Review, open Save confirmation. From diff/code, return to Review and open confirmation. | Never writes without the confirmation. Disabled when clean, read-only, invalid, or in Recovery; reason shown. |
| `q` | Main screen | Request close. Close immediately only when clean and no transaction is active. | Dirty: open unsaved dialog. Modal: cancel/close modal. Progress: do not interrupt critical phase. |
| `Esc`, `Ctrl-C` | Modal/search/input | Cancel the transient view and restore prior focus. | On a main screen, clear transient footer/search state and remain open. During a critical transaction, do not abort. |
| `ZZ` | All configurator buffers | Route to Review when dirty; close when clean. It never directly writes the scratch buffer. | During progress, unavailable. |
| `ZQ` | All configurator buffers | Open explicit discard confirmation when dirty; close when clean. | During progress, unavailable. |
| `Ctrl-W` window commands | Multiwindow code/diff | Retain normal Neovim window navigation. Closing one controlled pane returns to the prior configurator screen rather than leaving a broken layout. | No custom window command is required in single-window layouts. |

Additional rules:

- Printable keys have no mutation effect in Normal-mode main views unless listed.
- An input field is the only place that enters Insert mode. While it has focus, letters are text, not configurator commands.
- Confirmation dialogs are Normal-mode choice lists, not text inputs.
- The visible footer is authoritative for the current context; key help contains the complete table.
- Mouse clicks may place focus on a real actionable line if native mouse support is already enabled, but no flow may require a mouse.

## 8. Visual system and design tokens

### Spacing and structure

- Wide/normal outer horizontal margin: 2 cells. Narrow: 1 cell.
- Header: 2–3 rows; narrow: 2 rows.
- Footer: 1 row when it fits, otherwise 2 rows.
- One blank line separates conceptual groups. Do not put a blank line between every setting.
- Setting rows are one line wide/normal; a focused description may add one line. Narrow values occupy a second indented line.
- Use separators only between header/content/footer and between true panes. Avoid boxes around individual cards.
- Keep labels sentence case. Reserve uppercase for short status badges and safety states.
- Never center long paragraphs. Left-align instructional text and values.

### Focus and selection

- Focused row: `>` prefix plus `NvimConfiguratorSelectedRow` across the row.
- Selected single choice: `(x)`; unselected: `( )`.
- Enabled boolean: `[x]`; disabled boolean: `[ ]`.
- Focus and selection are intentionally separate. A focused unselected radio row displays `> ( )`.
- Disabled row: literal `[UNAVAILABLE]` or a written reason plus a muted highlight. It remains readable.
- Dirty setting: trailing `*`; header explains `* differs from saved` while any dirty row is visible.

### Semantic text symbols

ASCII is the primary system, not merely a fallback:

| Meaning | Symbol/text |
| --- | --- |
| Focus | `>` |
| Selected / unselected | `(x)` / `( )` |
| Enabled / disabled | `[x]` / `[ ]` |
| Dirty | `*` plus header text |
| Success/found | `[OK]`, `[DONE]`, `[SAVED]` |
| Warning | `[!]` or `[WARNING]` |
| Error | `[ERROR]`, `[FAILED]` |
| Missing dependency | `[MISSING]` |
| Optional dependency | `[OPTIONAL]` |
| Waiting/unknown | `[WAIT]`, `[CHECKING]`, `[UNKNOWN]` |
| Restart needed | `[RESTART]` |
| Read-only | `READ-ONLY` |
| Relationship | `->` |
| Added/removed diff | `+` / `-` |
| Border | `+`, `-`, `|` |

Do not require Nerd Fonts. Do not use icon font codepoints. Unicode checkmarks, arrows, rounded borders, and ellipsis may be offered only as an opt-in cosmetic layer and must not carry unique meaning.

### Semantic highlight groups

The plugin defines semantic groups with conservative links to established Neovim groups. User colorschemes may override them. No concrete foreground/background colors are hard-coded.

| Highlight group | Purpose | Suggested default link/attribute intent |
| --- | --- | --- |
| `NvimConfiguratorTitle` | Product and screen title | `Title` |
| `NvimConfiguratorSubtitle` | Product line and supporting text | `Comment` |
| `NvimConfiguratorHeading` | Group heading | `Statement` or bold normal text |
| `NvimConfiguratorText` | Primary prose | `Normal` |
| `NvimConfiguratorMuted` | Secondary descriptions | `Comment` |
| `NvimConfiguratorSelectedRow` | Keyboard focus | `CursorLine` plus visible `>` prefix |
| `NvimConfiguratorValue` | Draft value | `Identifier` |
| `NvimConfiguratorKey` | Footer key labels | `Special` |
| `NvimConfiguratorRecommended` | Recommendation label | `MoreMsg` |
| `NvimConfiguratorDisabled` | Unavailable row | `Comment` plus written reason |
| `NvimConfiguratorDirty` | Dirty marker/status | `WarningMsg` |
| `NvimConfiguratorRestart` | Restart-required text | `WarningMsg` |
| `NvimConfiguratorStatusOk` | Found/done/saved | `DiagnosticOk` with `[OK]` text |
| `NvimConfiguratorStatusInfo` | Built-in/optional/informational | `DiagnosticInfo` |
| `NvimConfiguratorStatusWarn` | Missing optional/pending | `DiagnosticWarn` |
| `NvimConfiguratorStatusError` | Blocking or failed state | `DiagnosticError` |
| `NvimConfiguratorPath` | Exact filesystem path | `Directory` |
| `NvimConfiguratorCode` | Generated Lua | normal Lua syntax groups |
| `NvimConfiguratorCodeAnnotation` | Setting provenance | `Comment` with `From:` prefix |
| `NvimConfiguratorBorder` | Pane/modal separators | `FloatBorder` or `WinSeparator` |
| `NvimConfiguratorModalTitle` | Modal heading | `FloatTitle` or `Title` |
| `NvimConfiguratorProgressCurrent` | Current transaction step | `CurSearch` plus `[....]` text |
| `NvimConfiguratorProgressPending` | Not-started step | `Comment` plus `[WAIT]` text |

Diff views should use built-in `DiffAdd`, `DiffDelete`, `DiffChange`, and `DiffText`, while retaining literal `+`/`-` prefixes in unified mode.

### Color-independent state rules

- Status always has a word, symbol, or both.
- Selected row always has `>`; selected value always has `(x)`/`[x]`.
- Warning/error prose names the consequence.
- Diff lines retain signs and headings.
- Restart state is written in the header and Review, not shown as a yellow value only.
- A temporary highlight may draw attention, but the cursor and text remain sufficient after it disappears.

### Generated-code annotations

Use three layers, in priority order:

1. Human-readable comments that will exist in generated Lua.
2. Preview-only source annotations beginning with `From:` when more precise mapping is useful.
3. Extmark highlights or virtual text as optional visual reinforcement.

No explanation may exist only as virtual text because plain terminal readers and some accessibility tools may omit it.

## 9. Accessibility and beginner usability

### No special font assumptions

ASCII is complete and is the default. Column calculations use display width rather than byte length. The product should remain legible with any monospaced terminal font.

### Color blindness and low-color terminals

Every state is textual. Validate the interface with `:set termguicolors?` both on and off, with a minimal colorscheme, and with highlights temporarily disabled. Focus, errors, success, diff direction, and disabled state must remain understandable.

### Screen readers and plain terminals

- Critical text is buffer content, not only virtual text, signs, popup titles, or statusline content.
- Cursor movement lands on the line that names the control.
- Screen changes place the cursor at a meaningful heading and send one concise route/status message.
- Avoid conceal, animated replacement, and status delivered only through ephemeral notifications.
- Provide a full-buffer Key help view that can be read linearly.
- Keep path lists and generated Lua copyable as ordinary text.

Neovim and terminal screen-reader support varies; v0.1 cannot guarantee a fully semantic GUI accessibility tree. It can guarantee linear text, stable cursor focus, no color-only meaning, and no extmark-only critical information.

### Vim notation

Footer labels use familiar literal names: `Enter`, `Esc`, `Space`, `Ctrl-C`, and `j/k`. Key help explains each without angle-bracket notation. Arrow keys work wherever `j`/`k` or `h`/`l` work, so knowing Vim motions is helpful but not required.

### Accidental keys

- `q` is harmless when a modal is open and guarded when dirty.
- `Esc` and `Ctrl-C` back out of transient UI; they do not discard a draft.
- During local writes, `q`, `Esc`, and `Ctrl-C` do not interrupt the transaction.
- `ZZ` and `ZQ` receive configurator semantics rather than treating the scratch buffer as an ordinary file.
- Native `:q` safety is reinforced by mirroring dirty state to the scratch buffer's modified flag.

### Plain language pattern

Each technical explanation follows the same short structure:

1. **What you get** — user-visible outcome.
2. **What this part does** — one or two sentences.
3. **Where it runs/lives** — built into Neovim, generated Lua, plugin, or external program.
4. **What happens if missing** — exact scope.
5. **Show technical name** — progressive disclosure.

Avoid tutorial paragraphs on primary screens. No primary description should exceed three lines at normal width.

## 10. Error and recovery design

### Error message anatomy

Every blocking error screen answers these questions in this order:

1. **What did not happen?** For example, `Configuration was not saved`.
2. **At which scope?** Local configuration, one plugin, one language feature, or recovery metadata.
3. **Did any file change?** `No files changed`, `Previous version restored and verified`, or `State incomplete`.
4. **Is the draft safe?** State the draft count and that it remains available.
5. **What can the user do now?** Put the safest actionable choice first.
6. **Where are technical details?** Keep traceback, command stderr, hashes, and temporary paths behind “Show technical details.”

Do not use an undifferentiated red banner labelled “Save failed.” The action scope matters more than the internal exception.

### Failure matrix

| Condition | When detected | Primary message | Disk guarantee | Draft | Primary actions |
| --- | --- | --- | --- | --- | --- |
| Generated Lua invalid | Draft generation or preflight | `Configuration was not saved — generated Lua is invalid` | No target writes started | Preserved | Open generated line; return to related setting; technical details |
| Target not writable | Preflight | `Configuration was not saved — target is not writable` | No target writes started | Preserved | Show path/permission guidance; recheck; close with draft guard |
| Write failed before replacement | Writing | `Configuration was not saved` | Existing target unchanged; temp may be cleaned | Preserved | Retry only after recheck; Review; details |
| Write failed after one or more replacements | Writing | `Configuration was not saved; restoration attempted` | Explicitly show restore verified or unverified | Preserved | If verified: Review; if unverified: Recovery |
| Final verification failed | Verifying | `Written files could not be verified` | Attempt restoration; report proof | Preserved | Recovery is primary if proof is incomplete |
| Required external program missing | Dependency check/preflight | `JavaScript tools cannot start` | Does not by itself prevent local config write | Preserved | Install guidance; recheck; save with feature pending; turn feature Off |
| Git missing | Dependency check/external phase | `Requested plugins cannot be downloaded` | Local configuration can still commit | Preserved then saved if local phase passes | Install Git; recheck/retry; choose None for optional picker |
| Plugin download failed | After local commit | `Configuration saved — plugin action needed` | Local generated files remain verified | Becomes persisted; plugin state pending | Retry plugin; details; change plugin choice in a new draft |
| Generated file manually changed | Ownership preflight | `Save stopped to protect a manually changed file` | No target writes started | Preserved | Compare; cancel; back up and explicitly replace |
| Transaction journal found | App open | `An earlier save did not finish` | Unknown until evidence check | Reconstruct if journal contains it; otherwise last draft not promised | Restore known-good; inspect; complete only if fully verifiable |
| Manifest corrupt/missing for generated-looking files | App open | `Ownership record cannot be verified` | Read-only; nothing changed | New edits disabled | Restore verified backup; inspect files; close |
| Backup restore fails | Recovery | `Recovery is incomplete` | Never claim last known-good | Recovery draft/journal preserved | Show exact files; details; do not resume normal saving |

### Generated Lua validation failure

The primary screen shows the generated relative file, line and column, a plain-language paraphrase, and the originating decision if known. Example structure:

```text
[ERROR] lua/user_config/editor.lua:12
The generated value for “Indentation width” is not valid Lua.

No configuration files were changed.
Your draft remains available.
```

“Open generated line” moves to a read-only code view with the diagnostic line focused. Do not expose a raw Lua stack trace until the user opens technical details.

### File-write failure

Multi-file writes are not globally atomic. The interface therefore treats journaling and verification as product behavior, not an implementation detail:

1. Validate all complete temporary contents.
2. Record intended old and new hashes in a transaction journal.
3. Create/verify a backup of any current managed files.
4. Replace targets.
5. Read back and verify every target.
6. Commit the new manifest.
7. Clear the transaction journal only after verification.

If a step fails, the UI lists all targets only in technical details; the primary view summarizes `0 files replaced`, `2 files restored and verified`, or `mixed state`. A mixed or unverified state automatically disables normal editing and saving until Recovery resolves it.

### Missing executable

“Required” is always qualified by its feature. Use `Required for Lua language tools`, `Required for Telescope native sorting`, or another exact feature name—never simply `Required` unless it blocks the entire product.

The policy is to allow a safe local configuration write with selected external features pending. The save confirmation must make this choice explicit. Generated startup behavior must tolerate unavailable plugins and external programs and report them through Health rather than raising opaque startup errors.

If product implementation cannot guarantee failure-tolerant setup for a selected feature, the policy must change to exclude that feature from the save. The confirmation names the omitted feature, mutates the draft only after confirmation, and updates Review before writing. It must never silently omit a selection.

### Plugin download failure

Plugin acquisition begins only after local configuration verification. Its result model is per plugin:

- `requested`
- `installing`
- `installed and verified`
- `failed`
- `pending because Git/network is unavailable`

If `nvim-lspconfig` fails, selected language tools become pending. If Telescope fails, only Telescope-backed search and picker actions become pending. If Neo-tree fails, only the optional file explorer becomes pending. The local success result remains visible. A retry reruns only the external plugin phase and never rewrites local configuration.

### Manual changes to generated files

Generated files are transparent and editable, so manual editing is legitimate. The configurator simply cannot overwrite it without consent.

When the disk hash differs from the manifest:

1. Label the disk file `Manually changed since last configurator save`.
2. Compare **last generated**, **current disk**, and **new proposal** as separate identities. Do not call the current disk file “saved configurator value.”
3. Disable ordinary Save.
4. Default action: Cancel and keep manual file.
5. Allow “Back up manual file and replace with proposed generated file” only after a second confirmation naming the absolute path.
6. Do not offer merge, parse-and-adopt, or selective import in v0.1.

The backup created before replacement is labelled `manual-before-replace`, not merely “previous,” so Recovery tells the truth about its origin.

### Incomplete transaction

On launch, inspect journal and hashes before rendering normal settings. Classify each target:

- matches last known-good;
- matches complete proposal;
- matches temporary content;
- missing;
- unknown.

Only two automatic conclusions are safe:

- If every intended target and manifest matches the complete proposal, verify again, clear the stale journal, and report that recovery completed.
- If every target matches the known-good set, clear safe temporary files after confirmation and report that the old configuration is intact.

Every mixed/unknown case opens Recovery and recommends restoring the last known-good backup. No background cleanup should erase evidence before the user can inspect it.

### Rollback

Rollback is a new local transaction, not a file copy shortcut:

1. Preview selected backup.
2. Explain which current generated files will be replaced.
3. Back up the current state as `before-rollback`.
4. Restore the chosen backup.
5. Validate Lua, hashes, and file presence.
6. Update manifest/history only after verification.
7. Show `[RESTART]` because the current process is not rewound.

If requested plugin sets differ, rollback of configuration does not automatically delete plugins from disk. Review must say that unused plugin data may remain and that removal is separate.

### Corrupted state

If manifest JSON is unreadable but configuration files still run, say exactly that: `Your Neovim configuration may still work; only configurator ownership cannot be verified.` Do not equate state corruption with config corruption.

If generated Lua is invalid on disk, offer the most recent verified backup. If no verified backup exists, remain read-only and show paths plus manual recovery guidance. Do not rebuild ownership from arbitrary disk files because that becomes existing-configuration import.

### Health screen

Health is the calm, later-use home for diagnostics. Its ordered checks are:

1. Neovim version and supported API availability.
2. Generated-file ownership and Lua validation.
3. Current-session versus configured values, with local-option scope noted.
4. External programs and what they affect.
5. Requested plugin presence and lockfile state.
6. Backup/journal status.

`r` rechecks non-mutating evidence. It does not install, repair, overwrite, or source configuration.

## 11. Implementation-oriented UI state model

### State sources and visible precedence

For each setting, keep distinct values rather than flattening them into one `value` field:

| Value | Meaning | Source | Normally visible? |
| --- | --- | --- | --- |
| Neovim default | Stock behavior for the supported Neovim version before user config | Versioned setting schema; never inferred from the current configured session | Details only |
| Persisted configured | Last value the configurator generated and verified | Manifest snapshot associated with file hashes | Row baseline indirectly; explicit in details |
| Draft | Proposed value in the open UI | In-memory draft state | Yes; this is the primary row value |
| Effective runtime | Value observed in the open Neovim process | Captured from the originating normal window/buffer or global option | Only when it differs, is unknown, or details is open |
| Validation | Whether the draft can be generated and saved | Schema and generation validation | Errors inline; otherwise Review/Details |
| Dirty | Whether draft differs from persisted | Derived comparison | `*` on row and count in header |
| Restart requirement | Whether a verified saved value is not active in the observed session | Derived from persisted/effective plus apply policy | Header/Overview/Review when true |
| Dependency status | Whether an external capability is found and verified | Recheck result with timestamp/path/version | On relevant feature, Dependencies, Review, Health |

The UI always treats Draft as the editable value. It never relabels Draft as Saved after a write starts; only the local commit event copies draft into persisted state.

### Local-option caveat

Some Neovim options are global, while others can vary per window, buffer, or filetype. Capture effective values from the user's originating normal editing window before opening the configurator. Label them `Current file before opening`, not simply `Current`, when scope is local.

The comparison status can be:

- `matches` — observed effective value equals persisted value;
- `differs` — observed value differs;
- `varies` — a filetype or local override makes one session-wide claim unsafe;
- `unavailable` — there was no appropriate originating normal buffer.

Overview may say `Current session matches` only when every relevant observable setting safely matches. Otherwise use `Current session differs` or `Some values vary by file`; details lists the evidence. Do not sample the configurator scratch window for user editor options such as wrapping or cursor line.

### Top-level controller state

```text
AppState
|- lifecycle: opening | ready | closing
|- mode: first_run | settings | readonly_existing | recovery
|- route: welcome | recipe | editor | languages | picker | review |
|          generated | diff | health | recovery | result | help
|- origin: tab/window/buffer identifiers and captured effective values
|- layout: width mode, dimensions, windows, scroll positions
|- focus: region, stable item ID, return stack
|- modal: none | explanation | confirmation | search | technical_details
|- schema: recipes, setting definitions, implications, dependency definitions
|- persisted: manifest snapshot or absent
|- draft: recipe provenance, explicit values, feature choices
|- history: undoable draft mutations
|- validation: per-item and aggregate results
|- dependencies: last check results
|- files: ownership and proposed file plan
|- transaction: local and external phase state
|- recovery: journal, backups, drift, recommended action
`- notifications: current footer message and message history
```

This is a conceptual shape, not a requirement to implement classes or a framework.

### Setting state

Each setting record needs:

- stable ID and beginner-facing label;
- group and order;
- value type and ordered allowed values;
- Neovim default plus supported-version provenance;
- recommended value per recipe;
- persisted value or `not_configured`;
- draft value;
- effective sample, scope, source buffer/window, and confidence;
- dirty status derived from persisted/draft;
- validation state: `unknown`, `valid`, `warning`, or `error`;
- generated file/line source mapping;
- implication relationships;
- restart status;
- short and expanded explanations.

Do not store `selected`, `saved`, `applied`, and `installed` as one boolean. They are separate facts.

### Feature state

Every feature backed by a plugin, parser, external tool, or runtime capability requires a richer record:

```text
desired: off | on
configured: absent | draft | persisted
dependencies: per-dependency statuses
plugin_request: none | draft | persisted
plugin_availability: absent | pending | installed | failed | unknown
runtime_availability: available | blocked | restart_needed | unknown
blocked_reason: plain-language scoped consequence
```

Examples:

- `desired=on`, `configured=draft`, server `missing`: selected, not written, not available.
- `desired=on`, `configured=persisted`, plugin `installed`, restart needed: written and installed, not active in this session.
- `desired=on`, local config committed, Telescope `failed`: written, external install failed, search feature unavailable.

### Dependency state

Each dependency record contains:

- stable ID and display name;
- technical command name;
- purpose in beginner language;
- `required_for` feature IDs;
- importance: product blocker, selected-feature blocker, or optional enhancement;
- check state: unknown, checking, found, missing, or check_error;
- resolved absolute path and version when found;
- checked-at timestamp;
- platform-specific guidance ID;
- whether saving config is allowed;
- exact unavailable capability.

Never turn `missing` into `found` because install guidance was opened or a command was copied.

### File plan and ownership state

Each planned path contains:

- absolute path and display-relative path;
- root category: config, plugin data, configurator state, or backup;
- action: none, create, modify, external-create, metadata-write, or backup;
- ownership: unmanaged, managed-matching, managed-drifted, new, or unknown;
- old hash, proposed hash, and disk hash where applicable;
- generated content identity and validation result;
- whether action belongs to local transaction or external plugin phase;
- backup destination if it will be replaced.

The Review renderer groups paths by action but retains absolute paths in an expandable view. The controller refuses to treat an unmanaged or unknown path as replaceable.

### Validation aggregate

Aggregate readiness has exactly three user-facing states:

- `BLOCKED` — the product cannot safely write local configuration.
- `READY WITH PENDING FEATURES` — local configuration can be safely written, but one or more selected external features will not be available.
- `READY` — local preflight passes and selected external prerequisites are currently present; plugin downloads can still fail later.

`READY` is not success. It is permission to show the save confirmation.

### Transaction state machine

```text
IDLE
  -> PREFLIGHT
       -> BLOCKED (no writes)
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
                           -> COMPLETE_WITH_PLUGIN_FAILURES
```

Key invariants:

- Draft mutation is frozen from confirmation until the local phase reaches a terminal state.
- `persisted = draft` only on `LOCAL_COMMITTED`.
- Dirty becomes false at local commit even if plugin acquisition later fails; plugin pending status remains separate.
- No `COMPLETE` state is possible before local verification.
- A local failure never clears the draft.
- A recovery-required outcome disables normal Save.
- Plugin retry starts at `EXTERNAL_INSTALLING`; it cannot re-enter local writing.
- Close is locked during local validating/backing-up/writing/verifying. After local commit, external work may expose an explicit safe close/cancel policy, but it cannot retroactively change local status.

### Events

| Event | Payload concept | State effect |
| --- | --- | --- |
| `APP_OPENED` | environment, dimensions, origin | Classify ownership and route to Welcome, Overview, read-only, or Recovery. |
| `RESIZED` | columns, rows | Recompute mode; preserve focused item ID and scroll as closely as possible. |
| `NAVIGATE_REQUESTED` | route/section ID | Validate availability, push return focus, change route. |
| `MOVE_FOCUS` | direction | Select next actionable item without changing values. |
| `RECIPE_SELECTED` | recipe ID | Create/update proposals and implications; mark provenance. |
| `RECIPE_REAPPLY_REQUESTED` | recipe ID, preservation choice | Preview bulk impact before changing manually customized values. |
| `VALUE_CYCLED` | setting ID, direction | Create one undoable draft mutation and revalidate affected output. |
| `VALUE_TOGGLED` | setting/feature ID | Toggle allowed value; recompute implications/dependencies. |
| `CHOICE_SELECTED` | group ID, choice ID | Change an explicit single choice; focus remains stable. |
| `DRAFT_UNDO_REQUESTED` | none | Reverse latest configurator mutation; never invoke text-buffer undo. |
| `DRAFT_RESET_REQUESTED` | reset target | Open confirmation; do not mutate yet. |
| `DRAFT_RESET_CONFIRMED` | reset target | Replace draft and add one grouped undo record. |
| `EXPLANATION_OPENED` | item ID | Open detail with return focus. |
| `SEARCH_OPENED` / `SEARCH_SUBMITTED` | scope/query | Render accessible results; selection navigates to stable item. |
| `GENERATED_VIEW_OPENED` | optional setting/file ID | Generate current draft preview and focus mapped line. |
| `DIFF_OPENED` | comparison identity | Build managed persisted-vs-draft or recovery comparison. |
| `DEPENDENCY_RECHECK_REQUESTED` | dependency IDs | Set checking; run non-mutating checks. |
| `DEPENDENCIES_RECHECKED` | results and timestamp | Update status; never infer install success from intent. |
| `REVIEW_REQUESTED` | none | Regenerate, revalidate, refresh file plan and dependencies, then route. |
| `SAVE_REQUESTED` | none | Run preflight; show blocked reasons or confirmation. |
| `SAVE_CONFIRMED` | acknowledged warnings | Freeze draft and start the journaled local phase. |
| `LOCAL_STEP_UPDATED` | step, status, path/count | Update progress without changing persisted values. |
| `LOCAL_CONFIG_COMMITTED` | manifest, hashes, backup | Set persisted snapshot, clear dirty, set restart state. |
| `LOCAL_SAVE_FAILED` | stage, error, restoration proof | Preserve draft; route to failure or Recovery. |
| `PLUGIN_INSTALL_STARTED` | plugin IDs | Begin separate external states after local commit. |
| `PLUGIN_INSTALL_RESULT` | per-plugin outcome | Update feature availability; choose complete/partial result. |
| `FILE_DRIFT_DETECTED` | path and hashes | Block ordinary Save; route to comparison/recovery actions. |
| `ROLLBACK_REQUESTED` | backup ID | Open path-specific confirmation. |
| `ROLLBACK_CONFIRMED` | backup ID | Start a new journaled local restore transaction. |
| `ROLLBACK_COMPLETED` | verification result | Update persisted/restart/recovery state only if verified. |
| `CLOSE_REQUESTED` | source key/command | Close when safe, show unsaved dialog, or refuse during transaction. |
| `DISCARD_DRAFT_CONFIRMED` | change count | Restore draft to persisted/absent and close; do not alter disk. |
| `SESSION_EFFECTIVE_REFRESHED` | captured option values | Update matches/differs/varies without mutating config. |

Prefer `LOCAL_CONFIG_COMMITTED` plus per-plugin results over a single broad `SAVE_SUCCEEDED` event. The broad event encourages exactly the false atomicity this product must avoid.

### Rendering rules

- Render from state; do not derive product truth from whichever text is currently in the scratch buffer.
- Make the buffer temporarily modifiable only while replacing its render; return it to read-only immediately.
- Preserve cursor by stable item ID across rerenders and width changes.
- Set the buffer modified flag from draft dirtiness after rendering, not from renderer writes.
- Keep all user mappings buffer-local and tear them down with the buffer/tab.
- Use a dedicated highlight namespace so rerendering or colorscheme changes can reapply semantic highlights without affecting user buffers.
- Treat `WinClosed`, `BufLeave`, `TabClosed`, and command-based quit as close requests when possible; do not depend only on the `q` mapping for draft protection.

## 12. Product design boundaries

The following ideas are attractive but should be explicitly rejected or postponed:

- **Large floating dashboard:** poor at narrow widths, awkward for long content, and visually detached from native buffer workflows.
- **Persistent three-pane layout at every width:** consumes space and creates focus complexity. Context is inline or on demand below 120 columns.
- **Live application of draft options:** makes selected, saved, and effective states harder to explain and can leave a partially changed session after cancel.
- **Unreviewed shell or tool installation:** crosses trust and platform boundaries. Mason, parser installation, plugin builds, and other external actions must be planned, shown, confirmed, and reported per item.
- **Generic remote plugin marketplace:** the product supports the audited catalog and a user-owned extension file, not arbitrary discovery and setup-schema inference.
- **Multiple simultaneous completion engines:** native completion and `blink.cmp` are supported choices, but generated configurations must select one coherent primary path.
- **AI and cloud sync:** outside the audited Kickstart scope and the product proof.
- **Existing-config import or Lua merge:** unsafe without parsing intent and ownership. Read-only exploration is the honest v0.1 behavior.
- **Automatic adoption of manually edited generated files:** would turn transparent code into an unreliable two-way editor.
- **Per-setting animated preview or dashboard artwork:** visual spectacle without decision value.
- **Nerd Font icons, rounded-border dependency, gradients, or animated spinners:** reduce portability and accessibility.
- **Custom colorscheme for the configurator:** semantic links to the user's scheme are sufficient and more native.
- **External terminal UI framework:** native Neovim primitives cover the bounded scope and avoid bootstrapping a dependency to configure dependencies.
- **Mouse-first controls:** native mouse focus may be a convenience, never a requirement.
- **Multi-select profiles:** confuses starting proposals with actual features. Keep one starting profile and progressive feature decisions.
- **Unreviewed bulk profile switching after onboarding:** risks overwriting deliberate settings. Preview all affected values and preserve manual changes by default.
- **Unvalidated free-form option input:** custom values are allowed only where the schema can validate type, range, encoding, and generated Lua.
- **Deletion of unused plugin data during configuration rollback:** plugin removal is not part of a local config rollback and is not atomic with it.

## 13. Usability-testing plan

Use Neovim 0.12 in an ordinary terminal. Recruit participants who know how to open and quit Neovim but have not built a Lua configuration. Do not teach the mappings before the task; allow the visible footer and `g?` to do that work. Record the screen, key sequence, verbal interpretation, wrong turns, and time to confidence—not only task completion.

### Task 1: Start a Web development setup

- **Scenario:** “You installed Neovim to work on a small TypeScript project. Begin configuration and choose the most suitable starting point.”
- **Success condition:** Participant explicitly selects Web development, can say that it is a starting profile rather than a permanent mode, and reaches Editor essentials.
- **Likely confusion:** Mistaking the focused `( )` row for an already selected option; thinking multiple profiles must be chosen.
- **Observe:** Whether `>` versus `(x)` is understood, whether “What will you edit first?” reduces hesitation, and time to Continue.

### Task 2: Change editor behavior and inspect its Lua

- **Scenario:** “Use four spaces and turn line wrapping on. Find the Lua caused by the wrapping choice.”
- **Success condition:** Participant changes only the two intended settings, identifies dirty markers, opens Generated Lua at the relevant setting, and understands that it is not saved or active yet.
- **Likely confusion:** Using `Space` on an enum, assuming preview applied the value, or getting lost returning from code.
- **Observe:** Key discovery source, errors recovered from, interpretation of `Draft preview | not written`, and focus restoration.

### Task 3: Explain one language-tool chain

- **Scenario:** “You want errors and suggestions in one selected language. Explain what Neovim, nvim-lspconfig, Mason, the external server, and the selected completion engine each do.”
- **Success condition:** Participant distinguishes built-in client behavior, setup defaults, an installer, the external program, and completion.
- **Likely confusion:** Believing `nvim-lspconfig` or Mason is the language server, or believing a setup ID is always the executable name.
- **Observe:** Which wording they repeat, where they open explanations, and any two concepts they still merge.

### Task 4: Respond to a missing dependency

- **Scenario:** Simulate one selected server or formatter as `[MISSING]`. “Find what is missing, what it affects, how it would be acquired, and whether you can still save editor settings.”
- **Success condition:** Participant names the missing program, identifies only the affected feature, finds installation or acquisition guidance, and uses Recheck rather than assuming success.
- **Likely confusion:** Treating `[MISSING]` as a total save blocker; expecting the UI to execute the installation command.
- **Observe:** Whether impact wording is read, whether copied guidance is mistaken for execution, and whether `[FOUND]` is awaited.

### Task 5: Review and predict the save

- **Scenario:** “Before saving, tell us every file category that may change, which plugins are requested, and when the settings become active.”
- **Success condition:** Participant finds exact paths, distinguishes local files from external downloads/records, notices implied `nvim-lspconfig`, opens Diff, and says restart is required.
- **Likely confusion:** Missing collapsed implied choices, thinking `nvim-pack-lock.json` is generated Lua, or assuming current-session application.
- **Observe:** Information retrieval order, any hidden group not discovered, and confidence before pressing Save.

### Task 6: Understand partial external failure

- **Scenario:** Complete a save with one requested plugin or tool download failing. “What succeeded, what failed, and what will happen if you restart now?”
- **Success condition:** Participant says local configuration is saved and verified, names only the unavailable feature, understands that restart remains safe, and knows retry does not rewrite config.
- **Likely confusion:** Reading the entire operation as failed or believing Retry could overwrite settings.
- **Observe:** First sentence the participant uses to summarize state and which action they choose.

### Task 7: Make one later change and attempt to quit

- **Scenario:** Reopen with `:Settings`, turn wrapping Off/On from its saved value, then press `q` before saving.
- **Success condition:** Participant lands on Overview, finds Editor, sees a one-setting diff, and uses Keep editing or Review and save from the guarded dialog without losing the draft accidentally.
- **Likely confusion:** Expecting `:Settings` to open the last onboarding step, misreading effective versus saved, or pressing `Esc` expecting discard.
- **Observe:** Time to setting, whether one-line diff builds confidence, and reaction to the safe default.

### Task 8: Protect an existing/manual configuration

- **Scenario:** First show an unmanaged non-empty config; then show a configurator-managed file that was manually edited. “Explore safely, and describe what you would do if you wanted the configurator's new version.”
- **Success condition:** Participant confirms nothing was modified in the unmanaged case; in the drift case they compare versions and understand that replacement first backs up the manual file.
- **Likely confusion:** Looking for a takeover/delete button, interpreting read-only as an error, or assuming a three-way merge exists.
- **Observe:** Trust response, whether the safety explanation feels respectful rather than blocking, and whether backup/replace consequences are understood.

### Test acceptance thresholds

Before implementation expands beyond the vertical slice:

- At least 4 of 5 beginners must distinguish focus from selection without coaching.
- At least 4 of 5 must state that Preview is neither Saved nor Active.
- At least 4 of 5 must distinguish local save success from plugin install failure.
- No participant should believe the existing-config screen modified, renamed, or deleted a file.
- No participant should lose a draft through a single `q`, `Esc`, or `Ctrl-C`.
- Median first-run completion when accepting a profile, excluding external installation time, should remain under eight minutes.

## 14. Final recommendation

### Recommended primary layout

Use a dedicated tab with a full-screen scratch-buffer workspace. At 120+ columns, use a restrained section/step rail, main content, and contextual details. At 80–119, use one main column with the focused explanation below its row. Below 80, stack labels and values and use expandable summary rows. Reserve floats for short explanations and confirmations. Use native split diff only when both panes remain readable; otherwise use unified diff.

### Recommended onboarding length

Use five visible stages after Welcome: Profile, Editor essentials, Development tools, Optional features, and Review. Only four screens contain choices, and the ordinary profile-acceptance path should take five to eight minutes. Generated Lua, dependency help, full section customization, and Diff remain available without becoming mandatory steps.

### Recommended default keyboard model

- `j`/`k` and arrows move.
- `h`/`l` and arrows change enum values.
- `Space` toggles/selects.
- `Enter` opens or activates the visible row.
- `?`/`K` explains; `g?` shows all keys.
- `p` shows generated Lua; `d` shows diff.
- `u` undoes one draft change; `U` confirms a full reset.
- `s` routes through Review and confirmation; it never writes immediately.
- `q` closes only when safe and otherwise opens the unsaved guard.

Keep every mapping buffer-local and show the current subset in the footer.

### Most dangerous usability risks

1. **Focus mistaken for selection.** Always show `>` separately from `(x)` or `[x]`.
2. **Draft mistaken for saved or active.** Persist `Draft`, `Saved`, `Current session`, and `[RESTART]` labels throughout the flow; never live-apply in v0.1.
3. **Plugin failure mistaken for config failure.** Separate local transaction and external plugin phases in state, progress, and result copy.
4. **nvim-lspconfig or Mason mistaken for the language server.** Explain each selected language stack in outcome-first order and explicitly say what each part is not.
5. **Kickstart optional examples mistaken for defaults.** Label upstream default, optional, conditional, and user-selected state separately.
6. **Unsafe overwrite of existing or manually changed files.** Manifest ownership and hash drift must block ordinary Save before any write.
7. **Critical state hidden by narrow layout or color.** Safety text stays literal and high in the reading order; below minimum size the form does not render.
8. **Generic `:Settings` collision.** Register the alias only when it is absent and never make it the sole documented entry point.

### First interface screen to prototype

Prototype **Editor behavior** first in wide, normal, and narrow modes. It exercises the core renderer, cursor-to-item mapping, focus versus selection, enum/toggle changes, dirty state, contextual explanations, generated-Lua source mapping, responsive reflow, footer discovery, and unsaved close guard. If this screen is not calm and comprehensible, the rest of the wizard will not be either.

### Smallest vertical slice that proves the design

Build one end-to-end path with no real plugin download yet:

1. Welcome.
2. Select General editing.
3. Edit only Line number style and Wrap long lines.
4. Open a setting explanation.
5. Preview the exact generated `init.lua` and `lua/user_config/editor.lua`.
6. Preview a new-file or one-line read-only diff.
7. Review exact absolute paths.
8. Run validation, backup/no-backup, write, and verification progress as distinct states.
9. Show Saved plus `[RESTART]` without applying the options live.
10. Reopen Overview, change one value, press `q`, and exercise the unsaved dialog.

This slice proves the product's central promise—understandable configuration as ordinary code, safely created inside Neovim—before LSP explanations, external dependency guidance, plugin acquisition, and recovery multiply the state space.
