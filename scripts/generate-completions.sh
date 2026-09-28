#!/bin/sh
set -eu
mkdir -p bin/completions
for shell in bash zsh fish powershell; do
  go run ./cmd/shoshinvim completion "$shell" > "bin/completions/shoshinvim.$shell"
done
