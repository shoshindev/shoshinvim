# Develop and verify

```sh
make check
make test                 # race detector and isolated real-Neovim tests
make cross                # all five target binaries
python3 scripts/test-installer.py
python3 scripts/terminal-smoke.py  # POSIX PTY; build bin/shoshinvim first
goreleaser check
goreleaser release --snapshot --clean --skip=publish,announce,validate
```

The real-Neovim suite downloads every offered plugin, theme, parser and language tool and checks each choice by what it does: `SHOSHINVIM_INTEGRATION=1 go test ./internal/engine/`. Languages whose runtime is missing are skipped; `SHOSHINVIM_ALL_LANGUAGES=1` makes that a failure. The **Real Neovim** workflow (`.github/workflows/integration.yml`) runs it on Linux with every runtime installed, weekly, on demand, and on pull requests labeled `integration`.

`scripts/editor-smoke.py` drives the real Neovim UI through MessagePack RPC against a separately installed target with Tokyo Night and a TypeScript language server, such as one accepting every recommendation. It requires Python `msgpack`, plus Pillow and fontTools for macOS screenshots, and never installs dependencies. Its `workspace.png` and `shortcuts.png` are the README's two editor images; pass a Nerd Font with `--font` when the target uses icons. The Lua integration script checks real language-server responses and formatting separately.

Cross compilation and local snapshot archives cover macOS ARM64/x86_64, Linux ARM64/x86_64, and Windows x86_64. CI is configured to run on each platform. Public installation channels are **not yet released**. The shell/PowerShell installers, Homebrew formula generator, checksums and draft-release provenance workflow are release infrastructure; do not advertise their remote installation URLs until release verification passes.

## Licenses

`THIRD_PARTY_NOTICES.md` lists the material ShoshinVim adapts and the license of every Go module compiled into the five release builds. It is generated; after changing dependencies, rewrite it and commit the result. CI fails when it is out of date.

```sh
make notices              # python3 scripts/third-party-notices.py
```

Release archives carry `LICENSE`, `NOTICE` and `THIRD_PARTY_NOTICES.md` next to the executable.

## README screenshots

The setup screens and the wordmark in the README are SVG files drawn from the interface itself. The screens show a dark terminal, the usual default, so they look the same on GitHub's light and dark pages; the wordmark uses the light-terminal colors, which keep enough contrast on both. Regenerate them after changing the interface:

```sh
make screenshots          # writes docs/assets/wordmark.svg and docs/assets/screens/
```

The review screen needs Neovim; without it, that screen is left as it is.

`docs/assets/social-preview.png` is the image GitHub shows when the repository is shared. It is drawn from `social-preview.html` in the same folder; after regenerating the screenshots, redraw it with Chrome and upload it under the repository's Settings → General → Social preview:

```sh
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless --hide-scrollbars \
  --window-size=1280,640 --screenshot="$PWD/docs/assets/social-preview.png" "file://$PWD/docs/assets/social-preview.html"
```

Headless Chrome can keep running after it writes the file; stop it once the PNG appears.

See [implementation status and remaining gates](implementation-status.md), [interface specification](interface-spec-v0.1.md), [configuration catalog](configuration-catalog.md), and [distribution specification](distribution-spec.md).
