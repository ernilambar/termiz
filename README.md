# Termiz

**One window. One shell. Nothing in the way.**

A minimal, GPU-accelerated terminal emulator for macOS. It gives you a real
shell on a PTY in a small native window — no browser engine, no tabs, no panes,
and one optional file to change the window, the font and the terminal.

## Install

Releases ship a single macOS arm64 app bundle; Intel Macs and universal
binaries are not supported. Pushing a `v*` tag publishes
`<Name>-macos-arm64.zip` and `SHA256SUMS`, where `<Name>` is the display name in
`appinfo.toml`.

```bash
unzip <Name>-macos-arm64.zip
xattr -dr com.apple.quarantine <Name>.app  # release bundle is ad hoc signed
mv <Name>.app /Applications/
open /Applications/<Name>.app
```

## Build and run

Requires Go 1.27.3+ and, on macOS, a C toolchain (the Metal backend is
Objective-C). Linux and Windows build cgo-free.

```bash
go build -o termiz .
./termiz
```

Or use the scripts. `dev.sh` builds the app bundle with go-gui's `buildapp`,
installs it to `/Applications`, installs a `termiz` command and relaunches the
app; `bundle.sh` runs the quality gate and produces the same archive the
release workflow publishes:

```bash
./scripts/dev.sh             # debug build, install app + CLI, relaunch
./scripts/dev.sh --release   # release build, verify arm64, install, relaunch
./scripts/bundle.sh          # gate, release build, dist/<Name>-macos-arm64.zip
```

The `termiz` command goes in `/usr/local/bin` when that directory is writable,
otherwise in `~/.local/bin`; set `TERMIZ_BIN_DIR` to pick another directory.
The script prints a `PATH` hint when the chosen directory is not on it.

With no arguments, `termiz` opens the window. The command line otherwise only
carries informational commands:

```bash
termiz --help      # or -h, or `help`
termiz --version   # or -V, or `version`
```

Run them as `./termiz …` to use the repository build instead. The version
printed is `version` in `main.go`, the same value the release tag must match
and the bundled app reports to its children.

## Configuration

The app reads one optional file: `$XDG_CONFIG_HOME/termiz/config.toml`, or
`~/.config/termiz/config.toml` when `XDG_CONFIG_HOME` is unset. It is read once
at startup, so a change applies on the next launch. Everything is optional; a
missing file means the defaults below. Unknown keys and out-of-range values are
logged and replaced by the default, so a typo never keeps the window from
opening.

```toml
[window]
width      = 1000
height     = 640
min_width  = 400
min_height = 240

[font]
family = "monospace"
size   = 14          # 6–72

[terminal]
scrollback_rows = 10000  # 0: go-term's 5000 rows, negative: no scrollback
cursor_blink    = true
shell           = ""     # empty: $SHELL, then the OS login shell
dir             = ""     # empty: home; ~ and $VAR expand
```

`terminal.shell` outranks `$SHELL`: a shell named in the file is the one the
window runs, and the child is given the matching `SHELL`. Leave it empty to
keep the `$SHELL` → OS login shell resolution.

Theme selection is a later phase.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow and the
checks a change must pass.
