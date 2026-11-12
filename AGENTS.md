# AGENTS.md

## Overview

`termiz` is a minimal terminal emulator: one native, GPU-accelerated window
running a real shell on a PTY. Go module `github.com/ernilambar/termiz`, with
`go-gui` for windowing/rendering and `go-term` for terminal emulation.

## Setup

Go 1.27.3+ (see `go.mod`) on macOS Apple silicon with a C toolchain: the Metal
backend is Objective-C, so cgo is mandatory. darwin/arm64 only.

```bash
go mod download
```

No env vars or secrets needed. Files the app uses:

- Config: `$XDG_CONFIG_HOME/termiz/config.toml` (default
  `~/.config/termiz/config.toml`), see `config.go`.
- State: `$XDG_STATE_HOME/termiz/state.toml` (default
  `~/.local/state/termiz/state.toml`), last shell directory, see `state.go`.
- Log: `~/Library/Logs/termiz.log` (`logfile.go`), since a GUI launch's stderr
  goes nowhere; truncated at launch past 1 MiB.

## Commands

```bash
go build -o termiz .
go test ./...
go vet ./...
gofmt -w .
./termiz
./scripts/dev.sh [--release]
./scripts/bundle.sh
```

## Conventions

- Everything stays in `package main`; this is a small single binary.
- **Shell:** `loginShell` uses `$SHELL`, then the OS login shell (`dscl`), and
  only an empty result defers to go-term's fallback. Never substitute
  `/bin/sh`. `terminal.shell` outranks `$SHELL`; that precedence lives in
  `shellSpec`. `-l` is passed (`shellArgs`) only for basenames in
  `loginShells`, since an unknown shell may reject it.
- **Config** is read, never written with values. `defaultConfig` is the only
  home of built-in values; `launch` takes every value from the loaded struct.
  Loading is forgiving: a missing file means defaults; a bad key or value is
  logged and defaulted (`applyKeys` decodes one key at a time); a syntax error
  ignores the whole file. `terminal.shell` resolves via `exec.LookPath`, then
  `loginPATH` (`resolveShell`), else the login shell. `config_file` loads a
  base file (relative to the including file); a missing target, cycle or chain
  past `maxConfigIncludes` is a warning.
- **Settings menu** (`menu.go`) opens the config in `$VISUAL`, `$EDITOR`, else
  the system text editor; terminal editors (`isTerminalEditor`) count as unset.
  The editor gets the shell's PATH (`loginPATH`, `userpath.go`: one `-i -l -c`
  probe with `WaitDelay`), since launchd's PATH misses editor CLIs. An
  unresolvable command (`commandResolves`) falls back to the text editor. Runs
  off the main thread. Tests replace `loginPATH`. A missing config is seeded
  with `configHeader` (comment only); an existing file is never touched.
  go-gui's app menu has no slot for custom items, so `installMenubar` declares
  a throwaway menu and `moveMenuItemsIntoAppMenu` (`menu_darwin.go`, cgo) moves
  its items above Quit. A native menubar needs App mode (`gui.NewApp` +
  `backend.RunApp`). `dev.sh` forwards `VISUAL`/`EDITOR` because `open(1)`
  drops the environment.
- **Reload** (Cmd+Shift+,) re-runs `loadConfig` off the main thread and applies
  it via `applyTermConfig` (`reload.go`); `configChanges` logs keys that need a
  restart. `window.confirm_on_quit` (off by default) shows `quitDialog` via
  `OnCloseRequest` while the shell is alive; a shell exiting on its own closes
  without a prompt.
- **Maximize:** `win.Maximize()` in `OnInit` (a zoom, not fullscreen). Live on
  reload via `Maximize()`/`Restore()`. No cgo of our own here.
- **Startup** lives in `gui.WindowCfg.OnInit`; failures log and close the
  window, never panic. `term.Term` is closed by the `defer` in `launch`, so
  closing the window tears down the PTY.
- Comment non-obvious platform or lifecycle behavior in place; no comments that
  restate code.
- **Identity:** `appinfo.toml` holds the app ID and display name, embedded by
  `main.go` and read by `cmd/buildapp`. `name` is the only spelling of the
  display name. No version in the manifest.
- **Icons** (`assets/`): `icon.svg` → `icon.png`, 1024×1024, full-bleed and
  opaque (macOS 26 masks it; transparency gets a gray plate). `buildapp` turns
  it into `termiz.icns`. `icon-dock.svg` → `icon-dock.png` (pre-rounded, 824px
  grid) is embedded as `WindowCfg.IconPNG`, because the Dock draws it unmasked.
  Re-render both at 1024×1024; keep the file names.
- **Theme:** `theme.name` matches go-term's corpus case-insensitively
  (`theme.go`), is passed as `Cfg.Themes[0]` so `COLORFGBG` agrees, and picks
  the chrome via `Theme.IsDark` (`chromeTheme`). Unknown names warn and fall
  back to `Default`. Never invent a palette; `termiz themes` is the list.
- **State** is written, never read as settings. `watchDir` samples the shell's
  directory every 2s, writing only on change; `rememberDir` flushes it and must
  run before `tm.Close()`. `terminal.dir` pins the start directory and disables
  writes. The directory is the OSC 7 report (`cwdLocalPath`, local host only),
  else the process cwd (`cwd_darwin.go`, libproc). Writes are atomic, 0600.
- New dependencies go through `go.mod`/`go.sum`.
- **go-term** is patched locally, never upstreamed: `go.mod` replaces it with
  `third_party/go-term`, generated from the tag in `UPSTREAM` plus `patches/`.
  Never edit the copy; change a patch and run `scripts/update-goterm.sh [tag]`.
  Test it with `cd third_party/go-term && go test ./...`.

## Release

The version lives only in `const version` in `main.go`.

1. Bump it, run the quality gate, commit `Bump version to X.Y.Z` on `main`.
2. `git tag -a vX.Y.Z -m "Tagging vX.Y.Z"`, push `main` and the tag.
   `release.yml` fails if the tag does not match `const version`, and publishes
   `Termiz-macos-arm64.zip` and `SHA256SUMS`.

## Quality gate

Each must exit 0 before work is complete; `gofmt -l .` must print nothing:

```bash
gofmt -l . && go vet ./... && go test ./... && go build -o termiz .
```

For UI/runtime changes, also run `./termiz` and confirm the window opens and
the shell responds.
