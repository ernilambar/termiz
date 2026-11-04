# AGENTS.md

## Overview

`terming` is a minimal terminal emulator: one native, GPU-accelerated window
running a real shell on a PTY. Core stack is Go (module
`github.com/ernilambar/terming`) with `go-gui` for windowing/rendering and
`go-term` for terminal emulation.

## Setup

Requires Go 1.27.3+ (see `go.mod`). On macOS a C toolchain is required because
the Metal backend is Objective-C; Linux and Windows build cgo-free.

```bash
go mod download
```

No environment variables, config files, or secrets are needed. The app reads
`$SHELL` at runtime; when it is unset the login shell is resolved from the OS.

## Commands

```bash
go build -o terming .
go test ./...
go vet ./...
gofmt -l .
gofmt -w .
go build ./...
./terming
```

## Conventions

- Keep everything in `package main` until there is a real reason to split
  packages; this is a small single-binary app, not a library.
- The shell resolution rule in `loginShell` is deliberate: `$SHELL` wins, then
  the OS login shell (`dscl` on macOS, `getent` on Linux), and only an empty
  result defers to `go-term`'s fallback. Never substitute `/bin/sh`, and never
  add platform branches without a matching `runtime.GOOS` guard.
- GUI startup lives inside `gui.WindowCfg.OnInit`; failures there log and close
  the window rather than panicking. `term.Term` is created there and closed via
  the `defer` in `main`, so closing the window tears down the PTY.
- Comment any non-obvious platform or lifecycle behavior in place (both
  `main` and `loginShell` carry such comments); do not add comments that merely
  restate the code.
- Do not introduce external dependencies without updating `go.mod`/`go.sum`
  and checking that non-macOS builds remain cgo-free.

## Quality gate

Run these in order and confirm each exits 0 before declaring work complete:

```bash
gofmt -l . && go vet ./... && go test ./... && go build -o terming .
```

`gofmt -l .` must print nothing. For UI/runtime changes, also launch the binary
with `./terming` and confirm the window opens and the shell responds.
