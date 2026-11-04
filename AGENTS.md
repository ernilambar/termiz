# AGENTS.md

## Overview

`termiz` is a minimal terminal emulator: one native, GPU-accelerated window
running a real shell on a PTY. Core stack is Go (module
`github.com/ernilambar/termiz`) with `go-gui` for windowing/rendering and
`go-term` for terminal emulation.

## Setup

Requires Go 1.27.3+ (see `go.mod`). On macOS a C toolchain is required because
the Metal backend is Objective-C; Linux and Windows build cgo-free.

```bash
go mod download
```

No environment variables or secrets are needed. The app reads `$SHELL` at
runtime; when it is unset the login shell is resolved from the OS. An optional
config file at `$XDG_CONFIG_HOME/termiz/config.toml` (default
`~/.config/termiz/config.toml`) sets the window, font, theme and terminal
values; see `config.go`. A written state file at
`$XDG_STATE_HOME/termiz/state.toml` (default `~/.local/state/termiz/state.toml`)
restores the last shell directory; see `state.go`.

## Commands

```bash
go build -o termiz .
go test ./...
go vet ./...
gofmt -l .
gofmt -w .
go build ./...
./termiz
./scripts/dev.sh
./scripts/dev.sh --release
./scripts/bundle.sh
```

## Conventions

- Keep everything in `package main` until there is a real reason to split
  packages; this is a small single-binary app, not a library.
- The shell resolution rule in `loginShell` is deliberate: `$SHELL` wins, then
  the OS login shell (`dscl` on macOS, `getent` on Linux), and only an empty
  result defers to `go-term`'s fallback. Never substitute `/bin/sh`, and never
  add platform branches without a matching `runtime.GOOS` guard. A
  `terminal.shell` in the config file outranks `$SHELL` by design; that
  precedence lives in `shellSpec`, not in `loginShell`.
- Configuration is read by the loader, never written. `defaultConfig` in
  `config.go` is the only place the built-in values live, and `launch` takes
  every window and terminal value from the loaded struct instead of a literal.
  Loading is forgiving on purpose: a missing file means defaults, and a
  malformed file, unknown key or out-of-range value is logged and replaced by
  its default rather than failing startup. A file that does not parse is
  ignored whole.
  `config_file` loads another file as the base, so settings can live outside
  `~/.config`: the including file is decoded on top, a relative target resolves
  against the including file's directory, and a missing target, a cycle or a
  chain deeper than `maxConfigIncludes` is a warning, not an error.
- `Termiz > Settings` (`menu.go`) opens the config in `$VISUAL`,
  then `$EDITOR`, and otherwise in the system text editor. The item belongs in
  the app menu, but go-gui's native app menu has no slot for app-defined items:
  `installMenubar` declares a throwaway top-level menu and
  `moveMenuItemsIntoAppMenu` (`menu_darwin.go`, cgo; `menu_other.go` is the
  stub) moves its items above Quit and removes the empty menu. Installing a
  native menubar needs App mode (`gui.NewApp` + `backend.RunApp`): only a
  registered App owns one. `open(1)` does not inherit the caller's environment,
  so the `dev.sh` CLI shim forwards `VISUAL` and `EDITOR`. The action creates
  the config directory and, for the system-editor fallback only, an empty
  config file (TextEdit refuses a missing path); it never changes values.
- Window maximization is macOS-only and lives behind `maximizeWindow`
  (`window_darwin.go`, cgo; `window_other.go` is the stub): go-gui exposes no
  maximize or fullscreen call, so the Darwin build zooms the AppKit key window
  in `OnInit`, and every other target keeps the configured size. Keep the two
  files in step, and never let the stub need cgo.
- GUI startup lives inside `gui.WindowCfg.OnInit`; failures there log and close
  the window rather than panicking. `term.Term` is created there and closed via
  the `defer` in `main`, so closing the window tears down the PTY.
- Comment any non-obvious platform or lifecycle behavior in place (both
  `main` and `loginShell` carry such comments); do not add comments that merely
  restate the code.
- `appinfo.toml` holds the app ID and display name: `main.go` embeds it for the
  running app and `cmd/buildapp` reads the same file when packaging the bundle,
  so the two identities cannot diverge. `name` is the only place the display
  name is spelled; the scripts and workflow read it from there. The release
  version comes from the git tag through `-version`; do not add a version to the
  manifest.
- The icon lives in `assets/`: `icon.svg` is the editable source and `icon.png`
  is the committed 1024×1024 full-bleed opaque render. macOS 26 and later mask
  it to the app-icon squircle; a transparent canvas makes the system draw its
  legacy gray plate behind the artwork instead. `appinfo.toml` names the PNG so
  `buildapp` converts it to `Contents/Resources/termiz.icns` (`sips` +
  `iconutil`) without an icon flag, and `main.go` embeds the same bytes as
  `gui.WindowCfg.IconPNG` because go-gui substitutes its own default Dock icon
  when that field is empty. Redraw the SVG and re-render the PNG at 1024×1024,
  keeping the canvas opaque; keep the file name, which the packaging path and
  the embed assume.
- Theming goes through go-term's bundled corpus: `theme.name` is matched
  case-insensitively by `theme.go`, the resolved theme is passed as
  `Cfg.Themes[0]` so the spawned child's `COLORFGBG` agrees, and the go-gui
  chrome is picked from `Theme.IsDark` (`chromeTheme`). A theme name the corpus
  does not know is a warning and falls back to `Default`; never invent a
  palette, and keep `termiz themes` the authoritative list.
- Session state is separate from config and written, never read as settings:
  `state.go` samples the shell's directory every 2s (`watchDir`, writing only
  on change, so an exiting shell or a crash still leaves the last directory)
  and flushes it in `rememberDir`, which must run before `tm.Close()` while the
  shell is still alive. `terminal.dir` pins the start directory and suppresses
  both writes. The directory is the shell's OSC 7 report (`cwdLocalPath`,
  percent-decoded) when it emits one, else the shell process cwd
  (`cwd_darwin.go` via libproc, `cwd_linux.go` via /proc, `cwd_other.go` stub).
  Keep the state write atomic and the file 0600.
- Do not introduce external dependencies without updating `go.mod`/`go.sum`
  and checking that non-macOS builds remain cgo-free.

## Quality gate

Run these in order and confirm each exits 0 before declaring work complete:

```bash
gofmt -l . && go vet ./... && go test ./... && go build -o termiz .
```

`gofmt -l .` must print nothing. For UI/runtime changes, also launch the binary
with `./termiz` and confirm the window opens and the shell responds.
