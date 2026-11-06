# Termiz

**One window. One shell. Nothing in the way.**

A minimal, GPU-accelerated terminal emulator for macOS. A real shell runs on a
PTY in a small native window rendered by go-gui and go-term — no browser
engine, no tabs, no panes, and one optional TOML file to change the window,
font, theme, and terminal.

## Quick start

Homebrew is the primary install; macOS on Apple silicon is required.

```bash
brew tap ernilambar/tap
brew trust ernilambar/tap
brew install --cask termiz
```

The cask installs `Termiz.app` and removes the quarantine flag the ad hoc
signature would otherwise leave on it. Update it with
`brew upgrade --cask termiz`.

For the release zip, a source build, the app-bundle scripts, and the `termiz`
command-line shim, see [Installing](docs/install.md).

## Usage

With no arguments, `termiz` opens the window on your login shell. The remaining
commands are informational:

```bash
termiz --help      # or -h, or `help`
termiz --version   # or -V, or `version`
termiz themes      # list the bundled color themes
```

The app menu holds `Settings…` (⌘,) and `Reload Configuration` (⌘⇧,). See
[Using termiz](docs/usage.md) for the window, menus, shell, session, and log.

## Configuration

Settings live in one optional file, `$XDG_CONFIG_HOME/termiz/config.toml`
(`~/.config/termiz/config.toml` when `XDG_CONFIG_HOME` is unset):

```toml
[window]
maximize = true

[font]
family = "Menlo"
size   = 12

[theme]
name = "Catppuccin Mocha"
```

Every key is optional; a missing file means the built-in defaults, and a
malformed value is logged and replaced rather than stopping startup.
`Termiz > Reload Configuration` re-reads the file without restarting the shell.
The full key reference, defaults, includes, and error behavior are in
[Configuration](docs/config.md).

## Documentation

- [Installing](docs/install.md) — Homebrew, releases, source builds, CLI shim.
- [Using termiz](docs/usage.md) — commands, menus, shell, session, log.
- [Configuration](docs/config.md) — every key, defaults, includes, errors.
- [Contributing](CONTRIBUTING.md) — workflow and the quality gate.

## License

[MIT](LICENSE) © Nilambar Sharma ([nilambar.net](https://nilambar.net/))
