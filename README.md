# terming

A terminal emulator built on [go-gui](https://github.com/go-gui-org/go-gui) and
[go-term](https://github.com/go-gui-org/go-term).

One window, one live shell on a PTY, rendered through go-gui's GPU backend.
The shell is `$SHELL`; when the environment carries no `SHELL` (a Finder launch),
the account's login shell is resolved instead of falling back to `/bin/sh`.
Closing the window tears down the shell, and the window closes when the shell
exits.

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
go build -o terming .
./terming
```

Or use the scripts. `dev.sh` builds the app bundle with go-gui's `buildapp`,
installs it to `/Applications` and relaunches it; `bundle.sh` runs the quality
gate and produces the same archive the release workflow publishes:

```bash
./scripts/dev.sh             # debug build, install to /Applications, relaunch
./scripts/dev.sh --release   # release build, verify arm64, install, relaunch
./scripts/bundle.sh          # gate, release build, dist/<Name>-macos-arm64.zip
```

Configuration and theme selection are a later phase; this is the live-terminal
foundation.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow and the
checks a change must pass.
