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

The real-Neovim suite downloads every offered plugin, theme, parser and language tool and checks each choice by what it does: `SHOSHINVIM_INTEGRATION=1 go test ./internal/engine/`. Languages whose runtime is missing are skipped; `SHOSHINVIM_ALL_LANGUAGES=1` makes that a failure. The **Real Neovim** workflow (`.github/workflows/integration.yml`) runs it on Linux with every runtime installed, weekly, on demand, and on pull requests labeled `integration`. Kotlin's case runs on the JDK that `SHOSHINVIM_KOTLIN_JAVA_HOME` names (Java 21 in the workflow), because kotlin-language-server fails to start on the Java 25 that jdtls needs.

`scripts/editor-smoke.py` drives the real Neovim UI through MessagePack RPC against a separately installed target with Tokyo Night and a TypeScript language server, such as one accepting every recommendation. It requires Python `msgpack`, plus Pillow and fontTools for macOS screenshots, and never installs dependencies. Its `workspace.png` and `shortcuts.png` are the README's two editor images; pass a Nerd Font with `--font` when the target uses icons. The Lua integration script checks real language-server responses and formatting separately.

`make cross` builds for macOS ARM64/x86_64, Linux ARM64/x86_64 and Windows x86_64, and CI runs on each of them. Releases ship only the four macOS and Linux archives. Windows stays out until the app and `scripts/shoshinvim-installer.ps1` have been tested on a real Windows machine.

## Releasing

Pushing a `v*` tag runs `.github/workflows/release.yml`. It runs CI on every platform, then GoReleaser builds the archives and `SHA256SUMS`, attaches the shell installer and records build attestations in a draft release. The `release` environment accepts only `v*` tags. Check the draft's files, then publish it. The installer and `shoshinvim update` read GitHub's latest release, which skips prereleases, so a tag with a suffix such as `-rc.1` is invisible to them. Published releases are immutable, so a broken release is fixed with a new version rather than replaced files.

## Licenses

`THIRD_PARTY_NOTICES.md` lists the material ShoshinVim adapts and the license of every Go module compiled into the macOS, Linux and Windows builds. It is generated; after changing dependencies, rewrite it and commit the result. CI fails when it is out of date.

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
