#!/usr/bin/env python3
"""Render a tap formula from the exact release checksum manifest. No publishing."""
import argparse
import re
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument('version')
parser.add_argument('checksums', type=Path)
args = parser.parse_args()
version = args.version.removeprefix('v')
if not re.fullmatch(r'\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?', version):
    parser.error('invalid version')
hashes = {}
for line in args.checksums.read_text().splitlines():
    fields = line.split()
    if len(fields) == 2:
        digest, name = fields
        if name in hashes or not re.fullmatch(r'[a-f0-9]{64}', digest):
            parser.error('invalid or duplicate checksum')
        hashes[name] = digest
print('class Shoshinvim < Formula')
print('  desc "Design and safely generate a Neovim configuration"')
print('  homepage "https://github.com/shoshindev/shoshinvim"')
print(f'  version "{version}"')
for os_name, suffix in [('macos', 'apple-darwin'), ('linux', 'unknown-linux-gnu')]:
    print(f'  on_{os_name} do')
    for arch, cpu in [('aarch64', 'arm'), ('x86_64', 'intel')]:
        asset = f'shoshinvim_{version}_{arch}-{suffix}.tar.gz'
        if asset not in hashes:
            parser.error(f'missing archive {asset}')
        print(f'    on_{cpu} do')
        print(f'      url "https://github.com/shoshindev/shoshinvim/releases/download/v{version}/{asset}"')
        print(f'      sha256 "{hashes[asset]}"')
        print('    end')
    print('  end')
print('''  def install
    bin.install "shoshinvim"
    bash_completion.install "completions/shoshinvim.bash" => "shoshinvim"
    zsh_completion.install "completions/shoshinvim.zsh" => "_shoshinvim"
    fish_completion.install "completions/shoshinvim.fish" => "shoshinvim.fish"
    man1.install Dir["man/*.1"]
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/shoshinvim --version")
  end
end''')
