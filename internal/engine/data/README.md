# Bundled metadata

`options.json` is a snapshot of `vim.api.nvim_get_all_options_info()` from Neovim
0.12.5. Names, types, scopes, defaults, abbreviations, and API flags come from that
runtime. The catalog adds editable aliases and validation; isolated Neovim probes
check selected values. It is not an exhaustive transcription of help restrictions.

Defaults that depend on the machine taking the snapshot — home and temporary
folders, install paths and the shell (`backupdir`, `backupskip`, `directory`,
`helpfile`, `packpath`, `runtimepath`, `shell`, `undodir`, `viewdir`) — are recorded
as `null`; a test keeps new snapshots from reintroducing them.

`plugins.json` pins the exact upstream commits resolved on 2026-09-28. A pin is
reproducibility metadata, not a claim that an upstream plugin has been audited.
Adapter support and release verification are tracked in `docs/implementation-status.md`.

Metadata changes must be reviewed alongside generated-output and integration tests.
They must never silently migrate persisted user choices.
