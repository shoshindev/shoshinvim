# Security policy

ShoshinVim is a development alpha. Security fixes go into the latest release only.

## Reporting a vulnerability

Report it privately on GitHub: open the [Security tab](https://github.com/shoshindev/shoshinvim/security) and choose **Report a vulnerability**. Please don't open a public issue for it.

Include the ShoshinVim version (`shoshinvim --version`), your operating system, and the steps that show the problem.

## What is in scope

ShoshinVim writes into your Neovim configuration folder, downloads plugins and language tools, and runs Neovim to check what it wrote. Problems on that path are in scope, for example:

- writing or deleting files outside the configuration folder you chose;
- running Lua from a configuration before you approved the review;
- downloading a plugin at a commit other than the one it pins;
- an installer that installs an archive whose checksum does not match `SHA256SUMS`.

Problems in a plugin or language tool itself belong with that project.
