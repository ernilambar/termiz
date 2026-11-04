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
`~/.config/termiz/config.toml`) sets the window, font and terminal values; see
`config.go`.

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
- Configuration is read, never written. `defaultConfig` in `config.go` is the
  only place the built-in values live, and `launch` takes every window and
  terminal value from the loaded struct instead of a literal. Loading is
  forgiving on purpose: a missing file means defaults, and a malformed file,
  unknown key or out-of-range value is logged and replaced by its default
  rather than failing startup. A file that does not parse is ignored whole.
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
- Do not introduce external dependencies without updating `go.mod`/`go.sum`
  and checking that non-macOS builds remain cgo-free.

## Quality gate

Run these in order and confirm each exits 0 before declaring work complete:

```bash
gofmt -l . && go vet ./... && go test ./... && go build -o termiz .
```

`gofmt -l .` must print nothing. For UI/runtime changes, also launch the binary
with `./termiz` and confirm the window opens and the shell responds.
