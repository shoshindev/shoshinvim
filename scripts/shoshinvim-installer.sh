#!/bin/sh
# Inspect before running. This installer never launches ShoshinVim or Neovim.
set -eu
repo=https://github.com/shoshindev/shoshinvim
tag=
install_dir=${SHOSHINVIM_INSTALL_DIR:-${XDG_BIN_HOME:-$HOME/.local/bin}}
force=false
fail() { printf '%s\n' "shoshinvim installer: $*" >&2; exit 1; }
while [ "$#" -gt 0 ]; do
  case "$1" in
    --version) [ "$#" -ge 2 ] || fail 'missing version'; tag=$2; shift 2 ;;
    --install-dir) [ "$#" -ge 2 ] || fail 'missing directory'; install_dir=$2; shift 2 ;;
    --force) force=true; shift ;;
    --no-modify-path) shift ;;
    --help) printf '%s\n' 'Usage: sh installer.sh [--version v0.1.0] [--install-dir PATH] [--force] [--no-modify-path]'; exit 0 ;;
    *) fail "unknown option: $1" ;;
  esac
done
case "$install_dir" in /*) ;; *) fail 'install directory must be absolute' ;; esac
# Receipts are JSON; reject unusual path characters rather than produce ambiguous ownership data.
case "$install_dir" in *'"'*|*'\'*|*'
'*) fail 'install directory contains unsupported characters' ;; esac
[ ! -e /etc/NIXOS ] || fail 'Use the Nix package/source path on NixOS.'
case "$(uname -s)/$(uname -m)" in
  Darwin/arm64) triple=aarch64-apple-darwin ;;
  Darwin/x86_64) triple=x86_64-apple-darwin ;;
  Linux/aarch64|Linux/arm64) triple=aarch64-unknown-linux-gnu ;;
  Linux/x86_64) triple=x86_64-unknown-linux-gnu ;;
  *) fail 'unsupported operating system or architecture' ;;
esac
if [ "$(uname -s)" = Linux ]; then
  ldd --version 2>&1 | head -n 1 | grep -Eqi 'glibc|GNU libc|GNU C' || fail 'Only glibc Linux is currently supported.'
fi
command -v curl >/dev/null 2>&1 || fail 'curl is required'
sha() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d ' ' -f 1; else shasum -a 256 "$1" | cut -d ' ' -f 1; fi; }
download() { curl --proto '=https' --proto-redir '=https' --tlsv1.2 --fail --location --silent --show-error "$1" -o "$2"; }
umask 077
tmp=$(mktemp -d "${TMPDIR:-/tmp}/shoshinvim-install.XXXXXXXX")
staged=
trap 'rm -rf "$tmp"; [ -z "$staged" ] || rm -f "$staged"' EXIT HUP INT TERM
if [ -z "$tag" ]; then
  resolved=$(curl --proto '=https' --proto-redir '=https' --tlsv1.2 --fail --location --silent --show-error -o /dev/null -w '%{url_effective}' "$repo/releases/latest")
  tag=${resolved##*/}
fi
printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$' || fail 'invalid release tag'
asset="shoshinvim_${tag#v}_${triple}.tar.gz"
download "$repo/releases/download/$tag/SHA256SUMS" "$tmp/SHA256SUMS"
download "$repo/releases/download/$tag/$asset" "$tmp/archive.tar.gz"
expected=$(awk -v name="$asset" '$2 == name { print $1 }' "$tmp/SHA256SUMS")
actual=$(sha "$tmp/archive.tar.gz")
[ "$expected" = "$actual" ] || fail 'archive checksum mismatch or ambiguous manifest; nothing installed'
tar -xzf "$tmp/archive.tar.gz" -C "$tmp" shoshinvim
[ -f "$tmp/shoshinvim" ] && [ ! -L "$tmp/shoshinvim" ] || fail 'archive has no regular shoshinvim executable'
dest=$install_dir/shoshinvim
state_dir=${XDG_STATE_HOME:-$HOME/.local/state}/shoshinvim
receipt=$state_dir/install.json
[ ! -L "$dest" ] && [ ! -L "$receipt" ] || fail 'refusing symlink destination/receipt'
if [ -e "$dest" ]; then
  [ "$force" = true ] || fail 'binary exists; use --force only for a script-managed installation'
  [ -f "$receipt" ] || fail 'existing executable has no receipt'
  old_path=$(sed -n 's/^  "path": "\(.*\)",$/\1/p' "$receipt")
  old_hash=$(sed -n 's/^  "sha256": "\(.*\)",$/\1/p' "$receipt")
  old_source=$(sed -n 's/^  "source": "\(.*\)",$/\1/p' "$receipt")
  [ "$old_path" = "$dest" ] && [ "$old_source" = script ] && [ "$old_hash" = "$(sha "$dest")" ] || fail 'receipt does not own the existing executable'
  cp "$dest" "$tmp/previous"
fi
mkdir -p "$install_dir" "$state_dir"
staged=$(mktemp "$install_dir/.shoshinvim.XXXXXXXX")
cp "$tmp/shoshinvim" "$staged"
chmod 755 "$staged"
binary_hash=$(sha "$staged")
cat > "$tmp/receipt.json" <<EOF
{
  "version": "$tag",
  "source": "script",
  "path": "$dest",
  "sha256": "$binary_hash",
  "archive_sha256": "$actual",
  "timestamp": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
}
EOF
receipt_tmp=$(mktemp "$state_dir/.receipt.XXXXXXXX")
cp "$tmp/receipt.json" "$receipt_tmp"
mv -f "$staged" "$dest"
staged=
if ! mv -f "$receipt_tmp" "$receipt"; then
  if [ -f "$tmp/previous" ]; then cp "$tmp/previous" "$dest"; chmod 755 "$dest"; else rm -f "$dest"; fi
  fail 'receipt replacement failed; prior installation restored'
fi
printf 'Installed %s to %s\n' "$tag" "$dest"
case ":$PATH:" in *":$install_dir:"*) ;; *) printf 'Add this directory to PATH: %s\n' "$install_dir" ;; esac
printf '%s\n' 'No shell startup files were modified. Run shoshinvim when ready.'
