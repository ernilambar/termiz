# Termiz

**One window. One shell. Nothing in the way.**

A minimal, GPU-accelerated terminal emulator for macOS. A real shell runs on a
PTY in a small native window rendered by go-gui and go-term — no browser
engine, no tabs, no panes, and one optional TOML file to change the window,
font, theme, and terminal.

## Quick start

The release bundle is macOS arm64 only, ad hoc signed, and needs one Gatekeeper
exception. `<Name>` is the display name in `appinfo.toml`:

```bash
unzip <Name>-macos-arm64.zip
xattr -dr com.apple.quarantine <Name>.app
mv <Name>.app /Applications/
open /Applications/<Name>.app
```

To build from source, install Go 1.27.3+ and the Xcode Command Line Tools
(`xcode-select --install`), then:

```bash
go build -o termiz .
./termiz
```

See [Installing](docs/install.md) for requirements, the app-bundle scripts, and
the `termiz` command-line shim.

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

- [Installing](docs/install.md) — releases, source builds, scripts, CLI shim.
- [Using termiz](docs/usage.md) — commands, menus, shell, session, log.
- [Configuration](docs/config.md) — every key, defaults, includes, errors.
- [Contributing](CONTRIBUTING.md) — workflow and the quality gate.

## License

[MIT](LICENSE) © Nilambar Sharma ([nilambar.net](https://nilambar.net/))
