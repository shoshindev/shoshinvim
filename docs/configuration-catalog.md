# ShoshinVim configuration catalog

- Status: authoritative product and configuration scope
- Target: Neovim 0.12+ with version-aware fallbacks where documented
- Catalog snapshot: 2026-09-28
- Interface: [conversational TUI specification](interface-spec-v0.1.md)

This catalog defines what the standalone ShoshinVim application must be able to represent, compare, explain, generate, validate, install, and recover. Its scope is Neovim itself plus the editor and IDE capabilities people commonly add through plugins and external tools.

No starter configuration is the product model. Profiles are editable proposals assembled from this catalog.

## 1. Catalog rules

1. **Choices before prescriptions.** When several maintained options solve the same problem, show the strongest general recommendation and the meaningful alternatives.
2. **Recommendations are contextual.** A choice may be recommended for balanced, minimal, maximum-feature, remote, or language-specific use. There is no universal winner.
3. **Recommendations are explainable.** Each mark stores rationale, tradeoffs, compatibility, external requirements, and the date it was reviewed.
4. **Recommendations never overwrite intent.** Applying or updating a profile previews every affected value and preserves manual choices by default.
5. **Native capability stays visible.** A plugin must not hide a good built-in Neovim option. “Built in” and “none” are valid choices.
6. **One owner per capability.** Completion, formatting, linting, file exploration, statusline, and similar families select one primary implementation unless combinations are known to compose safely.
7. **Configuration and acquisition are separate.** Generated Lua may be safely committed even when a plugin, parser, server, formatter, linter, test adapter, or debugger is unavailable.
8. **External actions require review.** Git clones, builds, package-manager commands, parser downloads, and tool installations are listed before confirmation and reported per item.
9. **Generated Lua is ordinary Lua.** It remains readable and functional without the configurator installed.
10. **Unmanaged configurations are read only.** Broad coverage does not authorize importing or rewriting arbitrary existing Lua.
11. **Catalog changes are migrations, not surprises.** Refreshing recommendations or plugin metadata never silently changes persisted user state.

Catalog labels:

- `[RECOMMENDED]` — balanced default for most users in the stated context.
- `[RECOMMENDED: MINIMAL]` — best fit when dependencies and surface area should stay small.
- `[RECOMMENDED: ADVANCED]` — best fit when control matters more than setup complexity.
- `[ALTERNATIVE]` — maintained option with a distinct and valid tradeoff.
- `[BUILT-IN]` — provided by Neovim without a plugin.
- `[EXTERNAL]` — requires a program or service outside Neovim.
- `[OPT-IN]` — disabled until explicitly selected because it adds cost, credentials, network access, or substantial complexity.
- `[EXPERIMENTAL]` — upstream stability or API guarantees require an explicit warning.

## 2. Recommendation and comparison model

Every replaceable capability records:

- stable capability and choice IDs;
- friendly name and repository/source;
- recommendation contexts and rationale;
- Neovim version range;
- plugin-manager and build compatibility;
- required and optional dependencies;
- feature coverage;
- configuration complexity;
- startup/runtime implications;
- accessibility and icon assumptions;
- conflicts and safe combinations;
- migration notes;
- last-reviewed date and evidence link.

`/compare <capability>` shows choices in a consistent table. The recommended mark must not be based only on popularity. An option may lose the mark while remaining supported; persisted configurations remain unchanged until the user chooses a migration.

## 3. Profiles

Profiles are starting proposals, never permanent modes.

### Balanced IDE `[RECOMMENDED]`

Sensible native behavior, Treesitter, LSP, diagnostics, completion, snippets, formatting, linting, fuzzy search, file navigation, Git signs, debugging hooks, test/task integration, and health checks. Only selected language toolchains are installed.

### Minimal native `[RECOMMENDED: MINIMAL]`

Native options, keymaps, commenting, completion, packages, LSP, diagnostics, quickfix, terminal, sessions, and spelling wherever Neovim already provides them. Plugins appear only when a selected capability has no adequate built-in path.

### Full IDE

Balanced IDE plus symbol outline, breadcrumbs, project/session restore, task runner, test UI, debugger UI, richer Git workflow, code actions, refactoring, database/HTTP tools, and optional AI assistance.

### Writing and notes

Prose-friendly wrapping, spelling/grammar, Markdown rendering, tables, link navigation, distraction reduction, and optional knowledge-base integrations.

### Remote and low-resource

Reduced animation, icons, background jobs, watchers, large previews, and network-managed features. Prefers built-in or minimal implementations and explicit loading.

### Language-focused

Starts from Balanced or Minimal, then applies one or more language recipes. Each recipe is independently editable.

### Custom

Starts with Neovim defaults and no optional plugins. The user selects capabilities individually.

## 4. Neovim core coverage

### Complete native-option registry

The configurator must expose every stable, user-settable option documented for the selected Neovim version, not only the curated choices below. The registry is versioned from Neovim help metadata and records:

- option name, short name, type, scope, and default;
- allowed values or validation rules;
- global, buffer-local, window-local, or global-local semantics;
- whether changing it is safe during the current session;
- modeline and security restrictions;
- related options and known invalid combinations;
- generated `vim.opt`, `vim.opt_local`, or API form;
- exact help tag and version provenance.

Read-only, terminal-derived, obsolete, or internal options are searchable documentation but not editable. Unknown custom values require schema validation; the interface never accepts arbitrary Lua as an option value.

### Guided editor behavior

| Capability | Recommended proposal | Alternatives |
| --- | --- | --- |
| Leader/local leader | Space | Any valid non-conflicting mapping |
| Line numbers | Absolute | Relative with absolute current line; Off |
| Mouse | All modes | Normal/Visual only; Off |
| Clipboard | System clipboard when available | Neovim registers only; custom provider |
| Search case | Smart case | Ignore case; exact case |
| Incremental search | On | Off |
| Search highlighting | On with clear action | Always on; Off |
| Live substitution | Split preview | Inline; Off |
| Wrapping | Filetype/profile-aware | Always on; always off |
| Break indent | On | Off |
| Scroll context | 8–10 lines | Any validated non-negative value |
| Horizontal scroll context | 5 columns | Any validated non-negative value |
| Split direction | Right and below | Left and/or above |
| Equalize resized splits | On | Off |
| Confirmation before abandon | On | Off |
| Update time | 250 ms | Custom positive duration |
| Mapping timeout | 300 ms | Custom positive duration |
| Persistent undo | On | Off; custom undo directory |
| Swap files | On for recovery | Off; custom directory |
| Backups | Off by default | On; writebackup only; custom directory |
| External file changes | Autoread with notification | Manual checktime |
| Session/history | ShaDa on | Reduced; custom item limits; Off |

### Indentation and whitespace

- Spaces, tabs, or filetype-specific indentation.
- Width, shift width, soft-tab width, smart tab, round indent, and backspace behavior.
- Automatic indentation: native, Treesitter/indent plugin, detected, or explicit.
- Indentation detection: `NMAC427/guess-indent.nvim` `[RECOMMENDED]`; `tpope/vim-sleuth` `[ALTERNATIVE]`; explicit only `[BUILT-IN]`.
- Visible whitespace, editable `listchars`, trailing-space policy, non-breaking spaces, and end-of-line markers.
- Color column: off, fixed columns, or filetype/text-width derived.
- Text width, format options, join behavior, virtual edit, and paste behavior.

### Files, buffers, windows, and tabs

- Encoding, file format, end-of-line, backup, swap, undo, autoread, autowrite, hidden buffers, and modified-buffer confirmation.
- Current working directory policy: global, tab-local, window-local, project-root, or unchanged.
- Split placement, resize behavior, minimum sizes, winbar/statusline/tabline visibility, last window behavior, and diff options.
- Buffer switching: native list, picker, bufferline, or tabline.
- Tab pages remain distinct from buffers; the UI explains the difference.
- Session save/restore includes buffers, tabs, windows, folds, cwd, terminal policy, and excluded paths.

### Command line, completion, spelling, and folds

- `wildmenu`, `wildmode`, `wildoptions`, command history, abbreviations, aliases, and command-line completion.
- Native insert completion sources, automatic completion, delay/timeout, popup behavior, and LSP completion.
- Spell languages, dictionaries, suggestions, capitalization, and per-filetype activation.
- Fold method: manual, indent, marker, expression, syntax, Treesitter expression, or LSP expression.
- Fold level, fold column, fold text, open/close behavior, and persistence.

### Keymaps, commands, and automatic behavior

The configurator supports:

- Normal, Insert, Visual, Select, Operator-pending, Command, Terminal, and language-map modes;
- global and buffer-local mappings;
- descriptions, nowait, silent, expression, remap, and replace-keycodes flags;
- conflict detection across generated mappings;
- mapping families and leader groups;
- user commands with arguments, ranges, completion, and descriptions;
- autocmds grouped by named augroup, event, pattern, buffer, nested, once, and description;
- filetype-specific options and actions;
- abbreviations and custom menu entries.

Advanced custom Lua belongs in the user-owned extension file and is never round-tripped through form fields.

## 5. Plugin and package management

| Choice | Recommendation | Best for | Tradeoff |
| --- | --- | --- | --- |
| Native `vim.pack` | `[RECOMMENDED: MINIMAL] [EXPERIMENTAL]` | Neovim 0.12+, direct Git sources, native lockfile | Still officially experimental; limited high-level lazy-loading policy |
| `folke/lazy.nvim` | `[RECOMMENDED]` | Mature dependency graph, lazy loading, rich UI and profiling | Adds a bootstrap dependency and manager-specific spec |
| `nvim-mini/mini.deps` | `[ALTERNATIVE]` | Small Lua-native dependency management | Fewer lifecycle/UI conveniences |
| Native packages/manual Git | `[ALTERNATIVE]` | Fully explicit expert-managed setups | Updates, dependencies, and lock state are manual |

Supported controls:

- source URL and display name;
- branch, tag, semantic range, or exact commit;
- required and optional dependencies;
- eager, event, command, filetype, key, or module loading where the manager supports it;
- conditional inclusion by OS, executable, environment, or feature;
- build/update hook with explicit shell review;
- lockfile and update policy;
- pin, update, rollback, retry, disable, and removal plan;
- requested, resolving, downloading, building, installed, update-available, failed, disabled, and unknown states.

The user selects the manager before plugin-backed features are generated. Changing managers is a reviewed migration, not a profile toggle.

## 6. Appearance and interface components

| Capability | Recommended | Alternatives |
| --- | --- | --- |
| Colorscheme | Built-in default until chosen | Tokyo Night; Catppuccin; Kanagawa; Rose Pine; Gruvbox variants; Nightfox; Everforest; any validated colorscheme |
| Icon provider | `nvim-mini/mini.icons` | `nvim-tree/nvim-web-devicons`; none |
| Statusline | `nvim-lualine/lualine.nvim` | `nvim-mini/mini.statusline` for minimal; `rebelot/heirline.nvim` for advanced; native format |
| Tab/buffer line | Native tabline or none | `akinsho/bufferline.nvim`; `romgrk/barbar.nvim`; custom heirline |
| Winbar/breadcrumbs | Off by default | `Bekaboo/dropbar.nvim`; `SmiteshP/nvim-navic`; custom native winbar |
| Indent guides | `lukas-reineke/indent-blankline.nvim` | `mini.indentscope`; `snacks.indent`; none |
| Scope/context | `nvim-treesitter/nvim-treesitter-context` | native winbar/breadcrumbs; none |
| Notifications | Native `vim.notify` | `folke/snacks.nvim` notifier; `rcarriga/nvim-notify` |
| Command/message UI | Native `[RECOMMENDED]` | `folke/noice.nvim` `[OPT-IN]` |
| Startup screen | None `[RECOMMENDED: MINIMAL]` | Snacks dashboard; `goolord/alpha-nvim`; `nvimdev/dashboard-nvim` |
| Cursor/scroll animation | None `[RECOMMENDED]` | `sphamba/smear-cursor.nvim`; `karb94/neoscroll.nvim` `[OPT-IN]` |

Appearance also covers background mode, transparency, true color, syntax/semantic highlight precedence, cursor shapes, sign definitions, fill characters, popup borders, floating-window style, concealment, and Nerd Font availability with plain-text fallbacks.

## 7. Editing ergonomics

| Capability | Recommended | Alternatives |
| --- | --- | --- |
| Commenting | Native `gc` `[BUILT-IN]` | `mini.comment`; `numToStr/Comment.nvim`; `tpope/vim-commentary` |
| Embedded-language comments | `JoosepAlviste/nvim-ts-context-commentstring` when needed | Filetype defaults only |
| Surroundings | `nvim-mini/mini.surround` | `kylechui/nvim-surround`; `tpope/vim-surround` |
| Text objects | `nvim-mini/mini.ai` | Treesitter textobjects; `wellle/targets.vim`; native only |
| Autopairs | `windwp/nvim-autopairs` | `mini.pairs`; none |
| Fast motion | `folke/flash.nvim` | `ggandor/leap.nvim`; `smoka7/hop.nvim`; native search |
| Multiple cursors | None `[RECOMMENDED]` | `jake-stewart/multicursor.nvim`; `mg979/vim-visual-multi` |
| Split/join structures | `Wansmer/treesj` | `AndrewRadev/splitjoin.vim`; language formatter |
| Alignment | Native/manual | `echasnovski/mini.align`; `junegunn/vim-easy-align` |
| Increment/decrement | Native `Ctrl-A/Ctrl-X` | `monaqa/dial.nvim` |
| Yank history | Native registers | `gbprod/yanky.nvim`; picker integration |
| Substitute/exchange | Native operators | `gbprod/substitute.nvim`; `tommcdo/vim-exchange` |

All plugin mappings are previewed, conflict-checked, and individually editable.

## 8. Search, navigation, files, and projects

### Picker

| Choice | Recommendation | Best for |
| --- | --- | --- |
| `folke/snacks.nvim` picker | `[RECOMMENDED]` | Broad modern source coverage and integrated UI |
| `ibhagwan/fzf-lua` | `[RECOMMENDED: MINIMAL]` | Fast picker with fzf semantics |
| `nvim-telescope/telescope.nvim` | `[ALTERNATIVE]` | Large extension ecosystem and familiar workflows |
| `nvim-mini/mini.pick` | `[ALTERNATIVE]` | Small integrated setup |
| Native commands/quickfix | `[BUILT-IN]` | No picker dependency |

Picker actions cover files, recent files, buffers, grep, current-buffer search, help, commands, keymaps, options, registers, marks, jumps, tags, diagnostics, quickfix, locations, symbols, references, implementations, Git files/branches/commits/status, colorschemes, projects, sessions, and configuration files.

### File explorer

| Choice | Recommendation | Best for |
| --- | --- | --- |
| `stevearc/oil.nvim` | `[RECOMMENDED]` | Editing directories like ordinary buffers |
| `nvim-neo-tree/neo-tree.nvim` | `[RECOMMENDED: SIDEBAR]` | Tree/sidebar workflows and multiple sources |
| `nvim-mini/mini.files` | `[ALTERNATIVE]` | Compact column-based navigation |
| `nvim-tree/nvim-tree.lua` | `[ALTERNATIVE]` | Traditional file tree |
| Native directory editing | `[BUILT-IN]` | Minimal dependency footprint |

### Projects, buffers, and sessions

- Project root: LSP root, Git root, marker files, cwd, manual choice, or per-language policy.
- Project selection: picker sources, recent directories, bookmarks, or custom roots.
- Sessions: native `:mksession` `[BUILT-IN]`; `folke/persistence.nvim` `[RECOMMENDED: SIMPLE]`; `rmagatti/auto-session` `[RECOMMENDED: AUTOMATIC]`; `stevearc/resession.nvim` `[ALTERNATIVE]`; `echasnovski/mini.sessions` `[ALTERNATIVE]`.
- Buffer cleanup: native delete; `snacks.bufdelete`; `mini.bufremove`.
- Bookmarks/marks: native marks, `ThePrimeagen/harpoon`, or `cbochs/grapple.nvim`.
- Large-file mode: threshold, disabled subsystems, syntax limit, preview limit, and warning policy.

## 9. Syntax, parsing, and code structure

### Treesitter

`nvim-treesitter/nvim-treesitter` is `[RECOMMENDED]` for installed languages, with native regex syntax as the fallback.

Configurable behavior:

- explicit parser list and version-aware parser availability;
- manual, prompt, or reviewed automatic installation;
- highlighting, indentation, folds, injections, locals, and queries;
- parser build tool and compiler checks;
- safe fallback when a parser is absent or broken;
- parser updates separate from local config writes.

Related choices:

- text objects: `nvim-treesitter/nvim-treesitter-textobjects` or `mini.ai`;
- context: `nvim-treesitter/nvim-treesitter-context`;
- autotag: `windwp/nvim-ts-autotag` or LSP/editor features;
- structural selection: incremental selection, textobjects, or native motions;
- playground/query inspection: built-in inspect tools and `:InspectTree`.

## 10. LSP and language intelligence

Neovim's native LSP client is always the execution layer.

| Capability | Recommended | Alternatives |
| --- | --- | --- |
| Server configurations | `neovim/nvim-lspconfig` | Handwritten `vim.lsp.config()` files `[BUILT-IN]`; language-specific client plugin |
| Tool acquisition | `mason-org/mason.nvim` with explicit review | System/package-manager tools; project-local tools; containers; manual paths |
| Mason/server bridge | `mason-org/mason-lspconfig.nvim` | Explicit executable mapping |
| Tool-list installation | `WhoIsSethDaniel/mason-tool-installer.nvim` | Mason UI/manual install; system bootstrap script |
| LSP progress | Native `vim.lsp.status()` | `j-hui/fidget.nvim`; statusline integration |

Every server record exposes command, environment, filetypes, root markers, workspace requirement, single-file behavior, settings, initialization options, capabilities, handlers, commands, logging, trace level, offset encoding, and workspace folders.

LSP features are independently configurable:

- definition, declaration, implementation, references, type definition;
- hover, signature help, code actions, rename, call/type hierarchy;
- document/workspace symbols and symbol outline;
- document highlights, semantic tokens, inlay hints, code lens;
- document/range/on-type formatting ownership;
- completion and resolve capabilities;
- workspace folders, file watchers, progress, and restart behavior;
- per-server feature disablement when another tool owns the capability.

Language-specific wrappers are offered when they add material functionality, including `folke/lazydev.nvim`, `mrcjkb/rustaceanvim`, `mfussenegger/nvim-jdtls`, `scalameta/nvim-metals`, `nvim-flutter/flutter-tools.nvim`, and `mrcjkb/haskell-tools.nvim`. The underlying native server configuration remains an alternative.

## 11. Completion and snippets

| Choice | Recommendation | Tradeoff |
| --- | --- | --- |
| `saghen/blink.cmp` | `[RECOMMENDED]` | Broad completion UX and sources; plugin/build compatibility must be checked |
| Native completion and LSP completion | `[RECOMMENDED: MINIMAL] [BUILT-IN]` | Smallest stack; fewer source/UI integrations |
| `hrsh7th/nvim-cmp` | `[ALTERNATIVE]` | Mature source ecosystem; more source plugins and configuration |
| No automatic menu | `[ALTERNATIVE]` | Manual completion only |

Configurable sources include LSP, buffer, path, snippets, command line, tags, dictionary, spell, Git, emoji, database, AI, and filetype-specific providers. Users choose source order, trigger policy, minimum prefix, fuzzy matching, documentation, signature help, ghost text, item layout, preselection, acceptance key, and fallback behavior.

Snippet choices:

- Blink's built-in snippet support `[RECOMMENDED]`;
- `L3MON4D3/LuaSnip` `[RECOMMENDED: ADVANCED]`;
- `nvim-mini/mini.snippets` `[ALTERNATIVE]`;
- native snippets/manual templates `[BUILT-IN]`;
- `rafamadriz/friendly-snippets` as an optional snippet collection.

Enter is never mapped to accept without an explicit warning and visible fallback behavior.

## 12. Diagnostics, lists, and code actions

Native `vim.diagnostic` is the source of truth. Configure virtual text, virtual lines, signs, underline, severity sorting/filtering, float border/header/source, update timing, jump behavior, and per-namespace overrides.

Presentation choices:

- native quickfix/location list `[BUILT-IN]`;
- `folke/trouble.nvim` `[RECOMMENDED]` for a unified diagnostics/references/list view;
- picker integration `[ALTERNATIVE]`;
- inline-only or signs-only minimal modes.

Code-action choices include native selection, preview/diff before applying workspace edits, lightbulb indicators, action-kind filtering, source actions on save, and explicit organize-imports/fix-all ownership.

## 13. Formatting and linting

### Formatting

| Choice | Recommendation | Best for |
| --- | --- | --- |
| `stevearc/conform.nvim` | `[RECOMMENDED]` | Multiple formatters, fallback policy, async/manual/save flows |
| Native LSP formatting | `[BUILT-IN]` | One trusted server owns formatting |
| `nvimtools/none-ls.nvim` | `[ALTERNATIVE: ADVANCED]` | Exposing CLI tools through LSP semantics |
| External command/task | `[ALTERNATIVE]` | Project-controlled scripts |

Support per-filetype formatter chains, first-available choices, working directory, environment, stdin/file mode, timeout, range formatting, embedded languages, format-on-save, format-after-save, manual-only, and LSP fallback/never/first/last policies.

### Linting

| Choice | Recommendation | Alternatives |
| --- | --- | --- |
| `mfussenegger/nvim-lint` | `[RECOMMENDED]` | LSP diagnostics only; none-ls; project task; no editor linting |

Configure filetype linters, multiple linters, stdin/file mode, cwd/env, events, debounce, modified-buffer policy, exit-code interpretation, parser, diagnostic namespace, and manual action.

Formatting, linting, and LSP diagnostics have separate ownership. Selecting two tools that produce the same result triggers a conflict explanation rather than silent duplication.

## 14. Git and collaboration

| Capability | Recommended | Alternatives |
| --- | --- | --- |
| Buffer hunks/blame | `lewis6991/gitsigns.nvim` | Native `:diffthis`; mini.diff |
| Git command workflow | `NeogitOrg/neogit` | `tpope/vim-fugitive`; terminal Git; LazyGit integration |
| Repository diff/history | Gitsigns diff or `sindrets/diffview.nvim` | Native diff; Fugitive views |
| GitHub issues/PRs | `pwntester/octo.nvim` `[OPT-IN]` | `gh` in terminal; browser |
| Conflict resolution | Native diff3 | `akinsho/git-conflict.nvim`; Diffview merge tools |

Configurable actions include hunk navigation/stage/reset/preview, buffer stage/reset, blame, word diff, deleted lines, revision comparison, commit log, status, branches, stash, push/pull, merge/rebase, conflict markers, and quickfix export.

Remote operations and authentication are never performed without an explicit command and review. Credentials are not written into generated Lua.

## 15. Debugging, testing, tasks, and terminals

### Debugging

- `mfussenegger/nvim-dap` `[RECOMMENDED]` as the DAP client.
- `rcarriga/nvim-dap-ui` plus `nvim-neotest/nvim-nio` for a full UI.
- `theHamsta/nvim-dap-virtual-text` for inline values.
- `jay-babu/mason-nvim-dap.nvim` or explicit adapter paths for acquisition/configuration.
- Vimspector `[ALTERNATIVE]` for users preferring its integrated workflow.

Support launch/attach configurations, arguments/env/cwd, breakpoints and conditions, log points, stepping, stack/scopes/watches, REPL, exception filters, run-to-cursor, adapter install state, and per-language recipes.

### Testing

- `nvim-neotest/neotest` `[RECOMMENDED]` with explicit language adapters.
- `vim-test/vim-test` `[ALTERNATIVE]` for command-oriented workflows.
- Project-native test commands `[BUILT-IN]` through terminal or task runner.

Support nearest/file/suite/last, watch mode, output, summary, diagnostics, coverage integration, debug-test handoff, and adapter health.

### Tasks and builds

- `stevearc/overseer.nvim` `[RECOMMENDED]` for templates, task state, dependencies, and output.
- Native `:make`, `makeprg`, `errorformat`, quickfix, and terminal `[BUILT-IN]`.
- `tpope/vim-dispatch` or `skywind3000/asyncrun.vim` `[ALTERNATIVE]`.
- Project scripts from npm, Make, Cargo, Go, Gradle, Maven, just, task, VS Code tasks, and custom commands.

### Terminal

- Native `:terminal` `[BUILT-IN]`.
- Snacks terminal/toggle `[RECOMMENDED]` when Snacks is selected.
- `akinsho/toggleterm.nvim` `[ALTERNATIVE]`.
- Shell, direction, size, persistence, cwd, environment, close behavior, and terminal mappings are configurable.

## 16. Symbols, refactoring, and navigation aids

| Capability | Recommended | Alternatives |
| --- | --- | --- |
| Symbol outline | `stevearc/aerial.nvim` | Trouble symbols; picker; `hedyhli/outline.nvim` |
| Breadcrumbs | `Bekaboo/dropbar.nvim` | `nvim-navic`; winbar text; off |
| Rename UI | Native rename | `smjonas/inc-rename.nvim` |
| Refactoring | Native LSP code actions | `ThePrimeagen/refactoring.nvim`; language-specific tools |
| References preview | Picker/Trouble | `dnlhc/glance.nvim`; location list |
| Code context | Treesitter Context | breadcrumbs; folds; off |

The product distinguishes protocol-standard actions from plugin-specific transformations and previews edits when the provider supports it.

## 17. Specialized IDE capabilities

These are optional topics, not default profile requirements.

| Area | Recommended option | Alternatives |
| --- | --- | --- |
| Database | `kristijanhusak/vim-dadbod-ui` + `tpope/vim-dadbod` | terminal clients; DB completion add-on |
| HTTP/API | `mistweaverco/kulala.nvim` | `rest-nvim/rest.nvim`; curl in terminal |
| Markdown rendering | `MeanderingProgrammer/render-markdown.nvim` | browser preview; plain Markdown |
| Markdown preview | `iamcco/markdown-preview.nvim` | live server/browser; none |
| LaTeX | `lervag/vimtex` | texlab-only; terminal build |
| Notes/wiki | `epwalsh/obsidian.nvim` | `zk-org/zk-nvim`; `nvim-neorg/neorg`; plain files |
| Grammar | Harper or LTeX LSP | built-in spell; LanguageTool integration |
| Containers | Docker/Compose LSP plus terminal tools | `lazydocker` terminal integration |
| Kubernetes | YAML/Helm LSP and schemas | `kubectl`/`k9s` terminal integration |
| REPL/notebooks | `Vigemus/iron.nvim` | `jupyter-vim`; Molten; terminal REPL |
| Remote development | SSH-launched Neovim | remote filesystem plugins; terminal multiplexer |

## 18. AI assistance `[OPT-IN]`

AI is part of the broad catalog but never part of a default profile. The recommendation is **Off** until the user explicitly chooses a provider, privacy policy, data scope, and credential source.

Choices:

- `olimorris/codecompanion.nvim` `[RECOMMENDED: PROVIDER-NEUTRAL]` for chat, inline workflows, multiple providers, and agent protocols;
- `avante-corp/avante.nvim` `[ALTERNATIVE]` for an IDE-like AI editing workflow;
- GitHub Copilot completion/chat plugins `[ALTERNATIVE]` for Copilot users;
- local-model integrations through Ollama-compatible providers `[ALTERNATIVE]`;
- no AI `[RECOMMENDED DEFAULT]`.

The configurator may generate plugin settings and environment-variable references. It must not collect, display, persist, or migrate API keys. Review discloses provider, network access, context sources, automatic actions, and files eligible to be sent.

## 19. Language and filetype recipes

Recipes are starting combinations. Every server, formatter, linter, debugger, test adapter, parser, and language-specific plugin remains independently selectable.

| Language/domain | Recommended intelligence | Alternatives | Format/lint/debug notes |
| --- | --- | --- | --- |
| Lua/Neovim | `lua_ls` + `folke/lazydev.nvim` | EmmyLua analyzer | Stylua; luacheck; local-lua-debugger |
| JavaScript/TypeScript | `tsgo`/official TypeScript server when project-compatible | `vtsls`; `ts_ls`; Deno separately | Biome or Prettier; ESLint/Biome; js-debug-adapter |
| Deno | `denols` | TypeScript server with Deno exclusions | `deno fmt`; `deno lint`; js-debug-adapter |
| HTML/CSS/JSON | VS Code language servers | Emmet LSP; Biome where supported | Prettier/Biome; stylelint; browser debugger |
| Tailwind CSS | `tailwindcss` | framework integration only | class sorting via Prettier plugin |
| Vue | `vue_ls` | vtsls Vue plugin combinations | Prettier/Biome; ESLint; browser debugger |
| Svelte | `svelte` | TypeScript-only fallback | Prettier Svelte; ESLint; browser debugger |
| Astro | `astro` | TypeScript/HTML fallbacks | Prettier Astro; ESLint |
| Angular | `angularls` | `ng.nvim` enhanced client | Prettier; ESLint; browser debugger |
| Python | BasedPyright | Pyright; ty; Jedi LSP | Ruff format or Black; Ruff; debugpy; neotest-python |
| Rust | `rust_analyzer` + Rustaceanvim | Native server config | rustfmt; Clippy; CodeLLDB; neotest-rust |
| Go | `gopls` | language-specific Go plugin | gofmt/goimports/gofumpt; golangci-lint; Delve; neotest-go |
| C/C++ | `clangd` | ccls | clang-format; clang-tidy; CodeLLDB/cpptools; CMake tools |
| CMake | `neocmake` or cmake-language-server | filetype support only | cmake-format/lint; build task integration |
| Java | `jdtls` + `nvim-jdtls` | native jdtls config | google-java-format; Checkstyle; java-debug; neotest-java |
| Kotlin | Kotlin language server | IntelliJ/external workflow | ktfmt; detekt; JVM debugger |
| C# | `roslyn_ls` | OmniSharp; csharp_ls | CSharpier/dotnet format; analyzers; netcoredbg |
| F# | FsAutoComplete/Ionide | native config | Fantomas; analyzers; netcoredbg |
| Ruby | Ruby LSP | Solargraph | RuboCop format/lint; rdbg; neotest-rspec |
| PHP | Intelephense | Phpactor | Pint/php-cs-fixer; PHPStan/Psalm; vscode-php-debug |
| Elixir | Expert or ElixirLS | Next LS | `mix format`; Credo; ExUnit adapters |
| Erlang | Erlang LS | lexical where supported | rebar3 format; Elvis; EUnit/Common Test |
| Gleam | Gleam language server | none | `gleam format/check/test` |
| Haskell | HLS + haskell-tools | native HLS config | Fourmolu/Ormolu; HLint; haskell-debug-adapter |
| Scala | Metals + nvim-metals | native Metals config | Scalafmt; Scalafix; JVM debugger |
| Zig | ZLS | none | `zig fmt`; compiler checks; CodeLLDB |
| Swift | SourceKit-LSP | Xcode external workflow | swift-format; SwiftLint; LLDB |
| Dart/Flutter | Dart LSP + flutter-tools | native Dart config | dart format/analyze; Dart debug/test |
| Bash/sh | bash-language-server | shellharden assistance | shfmt; ShellCheck; bash-debug |
| Fish | fish LSP where available | syntax/filetype only | fish_indent; fish checks |
| PowerShell | PowerShell Editor Services | filetype only | PSScriptAnalyzer; PowerShell debugger |
| SQL | SQLS or sql-language-server | database-client completion | SQLFluff; dialect-specific clients |
| JSON/YAML | jsonls/yamlls + SchemaStore | Biome for JSON | Prettier; yamlfmt; yamllint |
| TOML | Taplo | filetype only | Taplo format/lint |
| XML | lemminx | filetype only | xmllint; schema validation |
| Markdown | Marksman | Harper; LTeX | Prettier/mdformat; markdownlint; render/preview |
| LaTeX | texlab + VimTeX | texlab only | latexindent; ChkTeX; compiler/viewer selection |
| Terraform/OpenTofu | terraformls/tofu_ls | filetype only | terraform/tofu fmt; TFLint; Trivy |
| Docker/Compose | dockerls + compose LS | filetype only | dockerfmt where used; Hadolint |
| Nix | nixd | nil | nixfmt/alejandra; statix; deadnix |
| OCaml | ocamllsp | Merlin workflow | ocamlformat; dune tasks; test integration |
| Clojure | clojure-lsp | Conjure without LSP | cljfmt/zprint; clj-kondo; REPL tests |
| R | R language server + R.nvim | terminal R | air/styler; lintr; testthat |
| Julia | Julia LSP | terminal Julia | JuliaFormatter; static analysis; Julia debug/test |
| Protobuf | buf LS | protobuf-language-server | buf format/lint; protoc tasks |
| GraphQL | GraphQL LSP | framework plugins | Prettier; graphql-eslint |
| Prisma | Prisma language server | schema/filetype support | `prisma format/validate` |
| Ansible | ansible-language-server | YAML-only | ansible-lint; yamllint |
| Helm/Kubernetes | helm_ls + yamlls schemas | YAML-only | helm lint/template; kubeconform/kube-linter |

The catalog may add recipes without changing the schema. A recipe becomes recommended only after its commands, root rules, filetypes, tool ownership, and health checks are validated.

## 20. Dependencies and health

Base checks:

- supported Neovim version and build features;
- Git and selected plugin manager requirements;
- compiler/build tools, `make`, CMake, Rust, Node, Python, unzip/tar as selected plugins require;
- `rg`, `fd`, `fzf`, Tree-sitter CLI, clipboard provider, image protocol/tooling, and Nerd Font where selected;
- selected language runtimes and project package managers;
- servers, formatters, linters, debug adapters, test runners, task runners, database clients, and AI provider command/API environment.

Each dependency records purpose, provider, install scope, selected features, path, version, check command, status, last check, and whether absence blocks saving or only leaves a feature pending.

Installation providers may include Mason, system package manager, project-local package manager, language toolchain, container, manual path, or copyable instructions. The configurator never assumes these providers are interchangeable.

## 21. Generated file groups

The deterministic generator may emit only the modules required by selected state:

```text
init.lua
lua/user_config/core.lua
lua/user_config/options.lua
lua/user_config/keymaps.lua
lua/user_config/autocmds.lua
lua/user_config/appearance.lua
lua/user_config/plugins.lua
lua/user_config/navigation.lua
lua/user_config/editing.lua
lua/user_config/treesitter.lua
lua/user_config/lsp.lua
lua/user_config/completion.lua
lua/user_config/diagnostics.lua
lua/user_config/formatting.lua
lua/user_config/linting.lua
lua/user_config/git.lua
lua/user_config/debugging.lua
lua/user_config/testing.lua
lua/user_config/tasks.lua
lua/user_config/languages/<id>.lua
lua/user_config/specialized.lua
lua/user_config/ai.lua
lua/user_config/user.lua
.shoshinvim/state.json
.shoshinvim/manifest.json
<selected-manager-lockfile>
```

`user.lua` is created once and never overwritten. Empty modules are not generated. Optional feature modules must fail safely when a plugin or external tool is absent.

## 22. Conversational topic routing

`/configure <topic>` supports:

1. `profiles`
2. `core`
3. `options`
4. `keymaps`
5. `autocmds`
6. `appearance`
7. `editing`
8. `navigation`
9. `files`
10. `projects`
11. `sessions`
12. `plugins`
13. `treesitter`
14. `lsp`
15. `completion`
16. `diagnostics`
17. `formatting`
18. `linting`
19. `git`
20. `debugging`
21. `testing`
22. `tasks`
23. `terminal`
24. `languages`
25. `specialized`
26. `ai`
27. `dependencies`
28. `recovery`

`/compare <capability>` is the primary way to evaluate recommended and alternative implementations. `/search`, `/show`, `/review`, `/files`, `/lua`, `/diff`, `/health`, and `/recovery` provide discovery and inspection without requiring a navigation tree.

## 23. Extension boundary

Broad capability coverage does not mean pretending every plugin has a machine-readable setup schema.

- The curated catalog supports validated, versioned adapters for listed choices.
- A custom Git plugin can be added with source, version, dependencies, loading conditions, and a user-owned Lua configuration block.
- The configurator owns the plugin declaration only when explicitly requested; it never rewrites the user-owned block.
- Unsupported plugin settings remain ordinary Lua and are not represented as structured state.
- Curated adapters and recommendations can be updated independently from user state.

## 24. Explicit safety boundaries

The broad catalog still forbids:

- automatic import or merge of unmanaged Lua;
- automatic adoption of manually edited generated files;
- silent shell, Git, package-manager, parser, or network actions;
- credentials or tokens in generated configuration or state;
- live application of uncommitted draft settings;
- simultaneous conflicting owners for one capability without an explicit advanced override;
- deletion of plugin/tool data as part of configuration rollback;
- claiming support from a plugin name alone without a validated adapter;
- changing a persisted choice merely because the recommendation changed.

These boundaries keep the configurator broad without turning it into an unsafe plugin marketplace or an unreliable two-way Lua editor.

## 25. Primary references

The catalog snapshot uses first-party documentation as the baseline:

- [Neovim options](https://neovim.io/doc/user/options/)
- [Neovim LSP](https://neovim.io/doc/user/lsp/)
- [Neovim diagnostics](https://neovim.io/doc/user/diagnostic/)
- [Neovim packages and `vim.pack`](https://neovim.io/doc/user/pack/)
- [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig)
- [Mason](https://github.com/mason-org/mason.nvim)
- [lazy.nvim](https://github.com/folke/lazy.nvim)
- [blink.cmp](https://github.com/saghen/blink.cmp)
- [Conform](https://github.com/stevearc/conform.nvim)
- [nvim-lint](https://github.com/mfussenegger/nvim-lint)
- [nvim-dap](https://github.com/mfussenegger/nvim-dap)
- [Neotest](https://github.com/nvim-neotest/neotest)
- [Overseer](https://github.com/stevearc/overseer.nvim)
- [Gitsigns](https://github.com/lewis6991/gitsigns.nvim)

Individual catalog adapters must link their own upstream documentation and record when that evidence was last reviewed.
