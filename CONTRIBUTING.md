# Contributing

Contributions are welcome from anyone — bug reports, fixes, documentation, and
feature work. This guide covers setting up the project, making changes, and
running the checks required before a pull request.

## Who can contribute

Anyone. The project is maintained by
[Nilambar Sharma](https://nilambar.net/) ([@ernilambar](https://github.com/ernilambar)),
but external contributions are encouraged. If you are planning a larger change,
open an issue to discuss the approach first so we can agree on the direction
before you write code.

## Install

### Requirements

- Go 1.27.3 or later (see `go.mod`)
- macOS on Apple silicon, plus Xcode Command Line Tools —
  `xcode-select --install`. The Metal backend is Objective-C, so cgo is
  required on macOS. Linux and Windows build cgo-free, but releases target
  darwin/arm64 only.

### Set up

```sh
git clone git@github.com:ernilambar/termiz.git
cd termiz
go mod download
go build -o termiz .
```

## Fix and run

```sh
./termiz
```

This opens the native window with a live shell; it needs a graphical macOS
session.

For the app-bundle path — Dock icon, app menu, LaunchServices registration, and
the identity in `appinfo.toml` — build, install, and relaunch in one step:

```sh
./scripts/dev.sh             # debug build
./scripts/dev.sh --release   # release build, same flags as the release workflow
```

This replaces `/Applications/<Name>.app` and kills the running copy, so any
shell session in it is lost. It also installs a `termiz` CLI shim — in
`/usr/local/bin` when writable, else `~/.local/bin`, unless `TERMIZ_BIN_DIR`
says otherwise — so `termiz --help` and `termiz --version` work from a shell.
`./scripts/bundle.sh` runs the quality gate and builds the same archive a
release publishes, without installing anything:

```sh
./scripts/bundle.sh          # dist/<Name>-macos-arm64.zip + SHA256SUMS
```

The display name and bundle ID live in `appinfo.toml`. `main.go` embeds that
file for the running app and `cmd/buildapp` reads it when packaging, so the two
identities cannot drift. Do not add a version to the manifest: the app
version is `version` in `main.go`.

## Code layout

Everything is `package main` in `main.go` until there is a real reason to split
packages. `main` wires the window to `term.Term`; `loginShell` resolves the
shell and is the one place platform branches belong.

## Check locally before opening a pull request

Run these from the repository root and make sure they all pass:

```sh
gofmt -l .
go vet ./...
go test ./...
go build ./...
```

- `gofmt -l .` — must print nothing.
- `go vet ./...` — reports suspicious constructs.
- `go test ./...` — runs the test suite.
- `go build ./...` — confirms the app still compiles.

For UI or runtime changes, also launch the app and confirm the window opens and
the shell responds.

## Open a pull request

- Branch off `main`, push your branch, and open a pull request against `main`.
- Write a clear, concise commit message and pull request description explaining
  what changed and why.
- Make sure the checks above pass, and mention anything you could not verify
  locally.

## Release

Maintainers only. `version` in `main.go` is the single source of truth, and
the release workflow rejects a tag that does not match it. Bump the constant,
pass the quality gate, then tag and push:

```sh
# edit version in main.go
git tag v0.2.0
git push origin v0.2.0
```

The Release workflow builds `darwin/arm64`, packages the `.app` with
`cmd/buildapp`, and publishes `<Name>-macos-arm64.zip` and `SHA256SUMS` to a
GitHub Release. Intel Macs and universal binaries are not supported.

## Resources

- [go-gui documentation](https://github.com/go-gui-org/go-gui)
- [go-term](https://github.com/go-gui-org/go-term)
