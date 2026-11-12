# AGENTS.md

## Overview

`termiz` is a minimal terminal emulator: one native, GPU-accelerated window
running a real shell on a PTY. Core stack is Go (module
`github.com/ernilambar/termiz`) with `go-gui` for windowing/rendering and
`go-term` for terminal emulation.

## Setup

Requires Go 1.27.3+ (see `go.mod`) and macOS on Apple silicon with a C
toolchain: the Metal backend is Objective-C, so cgo is mandatory. termiz
targets darwin/arm64 only; there is no Linux or Windows code.

```bash
go mod download
```

No environment variables or secrets are needed. The app reads `$SHELL` at
runtime; when it is unset the login shell is resolved from the OS. An optional
config file at `$XDG_CONFIG_HOME/termiz/config.toml` (default
`~/.config/termiz/config.toml`) sets the window, font, theme and terminal
values; see `config.go`. A written state file at
`$XDG_STATE_HOME/termiz/state.toml` (default `~/.local/state/termiz/state.toml`)
restores the last shell directory; see `state.go`. A GUI launch's stderr goes
nowhere, so `openLog` (`logfile.go`) also writes the log to
`~/Library/Logs/termiz.log`, truncated at launch once it passes 1 MiB.

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
  the OS login shell (`dscl`), and only an empty result defers to `go-term`'s
  fallback. Never substitute `/bin/sh`. A
  `terminal.shell` in the config file outranks `$SHELL` by design; that
  precedence lives in `shellSpec`, not in `loginShell`. The shell is started
  with `-l` (`shellArgs`) so `~/.zprofile` runs on a Dock launch, but only
  when its basename is in `loginShells`; an unknown shell may reject the flag.
- Configuration is read by the loader, never written with values. `defaultConfig` in
  `config.go` is the only place the built-in values live, and `launch` takes
  every window and terminal value from the loaded struct instead of a literal.
  Loading is forgiving on purpose: a missing file means defaults, and an
  unknown key, a wrong-typed value or an out-of-range value is logged and
  replaced by its default rather than failing startup; `applyKeys` decodes one
  key at a time so a bad value costs only that key. A file with a syntax error
  is ignored whole. `terminal.shell` is resolved with `exec.LookPath`, then a
  bare name in `loginPATH` (`resolveShell`); a missing shell falls back to the
  login shell.
  `config_file` loads another file as the base, so settings can live outside
  `~/.config`: the including file is decoded on top, a relative target resolves
  against the including file's directory, and a missing target, a cycle or a
  chain deeper than `maxConfigIncludes` is a warning, not an error.
- `Termiz > Settings` (`menu.go`) opens the config in `$VISUAL`,
  then `$EDITOR`, and otherwise in the system text editor. A terminal editor
  (`isTerminalEditor`) counts as unset because the app has no TTY to give it.
  The editor runs with the PATH of the terminal's shell (`loginPATH`,
  `userpath.go`: one `-i -l -c` probe, marker-delimited output, `WaitDelay` so
  a daemon started by a profile cannot stall it), since launchd's PATH misses
  editor CLIs. `commandResolves` lets `sh` parse the command, so quoted paths
  work; one that still does not resolve falls back to the text editor. The
  action runs off the main thread. Tests replace `loginPATH` and never run the
  developer's shell.
  The item belongs in
  the app menu, but go-gui's native app menu has no slot for app-defined items:
  `installMenubar` declares a throwaway top-level menu and
  `moveMenuItemsIntoAppMenu` (`menu_darwin.go`, cgo) moves its items above Quit and removes the empty menu. Installing a
  native menubar needs App mode (`gui.NewApp` + `backend.RunApp`): only a
  registered App owns one. `open(1)` does not inherit the caller's environment,
  so the `dev.sh` CLI shim forwards `VISUAL` and `EDITOR`. The action creates
  the config directory and seeds a missing file with `configHeader`, a comment
  that links `docs/config.md` and sets no value (the system editor also needs
  the path to exist); an existing file is never touched.
- `Termiz > Reload Configuration` (Cmd+Shift+,) re-runs `loadConfig` off the
  main thread and applies the result through `applyTermConfig` (`reload.go`),
  which uses go-term's live setters. `configChanges` names the keys a running
  window cannot adopt, so they are logged as needing a restart. Closing the
  window or quitting while the shell is alive shows `quitDialog` via
  `WindowCfg.OnCloseRequest` when `window.confirm_on_quit` is true (off by
  default); a shell
  exiting on its own calls `Window.Close` directly and never prompts.
- Window maximization is `win.Maximize()` in `OnInit` when `window.maximize`
  is true. go-gui zooms rather than entering fullscreen, so the title bar and
  traffic lights stay; termiz keeps no cgo of its own. The key is live: a
  reload calls `win.Maximize()` or `win.Restore()`, which returns the frame
  the window was zoomed from (the configured size).
- GUI startup lives inside `gui.WindowCfg.OnInit`; failures there log and close
  the window rather than panicking. `term.Term` is created there and closed via
  the `defer` in `launch`, so closing the window tears down the PTY.
- Comment any non-obvious platform or lifecycle behavior in place (both
  `main` and `loginShell` carry such comments); do not add comments that merely
  restate the code.
- `appinfo.toml` holds the app ID and display name: `main.go` embeds it for the
  running app and `cmd/buildapp` reads the same file when packaging the bundle,
  so the two identities cannot diverge. `name` is the only place the display
  name is spelled; the scripts and workflow read it from there. The version is
  `const version` in `main.go`, passed to buildapp as `-version`; the release
  workflow fails when the tag does not match it. Do not add a version to the
  manifest.
- The icon lives in `assets/`: `icon.svg` is the editable source and `icon.png`
  is the committed 1024×1024 full-bleed opaque render. macOS 26 and later mask
  it to the app-icon squircle; a transparent canvas makes the system draw its
  legacy gray plate behind the artwork instead. `appinfo.toml` names the PNG so
  `buildapp` converts it to `Contents/Resources/termiz.icns` (`sips` +
  `iconutil`) without an icon flag. `main.go` embeds `icon-dock.png` (from
  `icon-dock.svg`: the same artwork pre-rounded on the 824px grid) as
  `gui.WindowCfg.IconPNG`, because the Dock draws that image unmasked and go-gui
  substitutes its own default Dock icon when the field is empty. Redraw both
  SVGs and re-render both PNGs at 1024×1024, keeping `icon.png` opaque; keep
  the file names, which the packaging path and the embed assume.
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
  percent-decoded, a remote host ignored) when it emits one, else the shell process cwd
  (`cwd_darwin.go` via libproc).
  Keep the state write atomic and the file 0600.
- Do not introduce external dependencies without updating `go.mod`/`go.sum`.
- go-term is patched locally and never sent upstream: `go.mod` replaces it
  with `third_party/go-term`, a generated copy of the tag in
  `third_party/go-term/UPSTREAM` plus `third_party/go-term/patches`
  (`git format-patch` output: `font.line_height`, and box drawing, block
  elements and Powerline U+E0B0–E0BF drawn as cell geometry). Never edit the
  copy; change a patch and run `scripts/update-goterm.sh [tag]`, which also
  moves to a new tag. The nested `go.mod` keeps it out of `go test ./...`; test
  it with `cd third_party/go-term && go test ./...`.

## Quality gate

Run these in order and confirm each exits 0 before declaring work complete:

```bash
gofmt -l . && go vet ./... && go test ./... && go build -o termiz .
```

`gofmt -l .` must print nothing. For UI/runtime changes, also launch the binary
with `./termiz` and confirm the window opens and the shell responds.
