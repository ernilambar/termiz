# Termiz

**One window. One shell. Nothing in the way.**

A minimal, GPU-accelerated terminal emulator for macOS. It gives you a real
shell on a PTY in a small native window — no browser engine, no tabs, no panes,
and one optional file to change the window, the font, the theme and the
terminal.

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
termiz themes      # list the bundled color themes
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
opening. The app writes one file of its own, the state file described under
`terminal.dir`, and never touches the config.

```toml
config_file = "~/dotfiles/termiz/config.toml"  # optional: load this first

[window]
maximize   = true   # fill the screen's visible area on launch
width      = 1000
height     = 640
min_width  = 400
min_height = 240
padding    = 8       # 0–64; inset from the window edge to the terminal

[font]
family = "Menlo"
size   = 12          # 6–72

[theme]
name = "Catppuccin Mocha"  # any bundled theme; empty: go-term's default

[terminal]
scrollback_rows = 10000  # 0: go-term's 5000 rows, negative: no scrollback
cursor_style    = "block"  # block, underline or bar
cursor_blink    = true
cursor_lock     = false  # ignore a child's cursor-shape requests
shell           = ""     # empty: $SHELL, then the OS login shell
dir             = ""     # empty: last session's directory, then home
```

The cursor keys select the shape the window starts with — `block`,
`underline`, or `bar` (case-insensitive, and `beam` is accepted for `bar`) —
whether it blinks, and whether the window obeys the child's own cursor
requests. With `cursor_lock` off, which is the default, an editor can switch
to a bar for insert mode (vim does); setting it to `true` pins the configured
shape and blink for the life of the window. `reset` returns to the configured
cursor either way. An unknown style is logged and replaced by `block`.

`window.maximize`, on by default, opens the window filling the screen's
visible area — the same as the Window > Zoom menu item. Set it to `false` to
open at `width`×`height` instead. It is macOS-only: a Linux or Windows build,
or a cgo-free Darwin build, keeps the configured size.

`window.padding` is the inset, in points, between the window edge and the
terminal, and defaults to `8`. The terminal keeps a 4 px inset of its own
inside it, so text starts `padding + 4` points from each edge; `0` is allowed
and leaves only that inner inset. Values outside `0`–`64` are logged and
replaced by `8`.

`config_file` lets a config live anywhere — a dotfiles repository, for
instance — while the file at the path above stays a one-line pointer. The named
file is loaded first and the file holding the key is applied on top, so the
pointer's own values win if it sets any. A relative path resolves against the
directory of the file that names it, and `~` and `$VAR` expand. A missing
target, a cycle, or a chain more than 8 files deep is logged and skipped; the
rest of the file still applies.

`Termiz > Settings…` (⌘,) runs `$VISUAL`, then `$EDITOR`, and opens
this file with it; when neither is set, or the editor is a terminal one such as
vim or nano, the default text editor opens it. The
directory is created when missing, and a missing file is created with a comment
pointing at `docs/config.md`. The app never writes a setting, so the editor is
the one that changes the contents.

`terminal.shell` outranks `$SHELL`: a shell named in the file is the one the
window runs, and the child is given the matching `SHELL`. Leave it empty to
keep the `$SHELL` → OS login shell resolution. On macOS the shell starts as a
login shell, so `~/.zprofile` (Homebrew's `PATH` setup) runs.

With `terminal.dir` empty the shell starts where the previous session ended,
so quitting and reopening resumes in the same directory. termiz records the
shell's directory while the session runs (a 2-second sampler that writes only
when it changes, and again on close) in
`$XDG_STATE_HOME/termiz/state.toml` (`~/.local/state/termiz/state.toml` when
`XDG_STATE_HOME` is unset) and restores it on the next launch. Sampling is what
keeps the directory when the shell exits on its own or the app is force-quit.
The directory comes from the shell's OSC 7 report when it emits one — most
shell integrations do; a report from a remote host over ssh is ignored — and
otherwise from the shell process itself, so no
shell setup is required. A first launch, or a saved directory that no longer
exists, falls back to home. Naming `terminal.dir` pins the start directory:
that value wins and the state file is not rewritten.

`theme.name` selects one of the 602 color themes go-term bundles (the
iTerm2-Color-Schemes corpus), matched case-insensitively so `catppuccin mocha`
and `Catppuccin Mocha` are the same theme; `termiz themes` prints the valid
names, `Default` first. Empty, or `Default`, keeps go-term's built-in dark
theme. The window chrome follows the chosen theme's light or dark character, so
a light theme gets a light title bar and focus ring. An unknown name is logged
and the default theme is used; there is no fallback to a half-applied palette.

```bash
termiz themes | grep -i catppuccin
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow and the
checks a change must pass.

## License

[MIT](LICENSE) © Nilambar Sharma ([nilambar.net](https://nilambar.net/))
