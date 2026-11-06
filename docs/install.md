# Installing termiz

## Supported platforms

termiz targets macOS on Apple silicon only. The Metal backend is Objective-C,
so cgo is mandatory, and the release workflow builds `darwin/arm64` alone.
Intel Macs and universal binaries are not supported; the scripts refuse to
build on an Intel host.

Running the bare binary needs Go 1.27.3+ and the Xcode Command Line Tools
(`xcode-select --install`). The app bundle additionally needs the identity in
`appinfo.toml`; its `name` is the display name used everywhere below as
`<Name>`.

## Install with Homebrew

The primary install is the `termiz` cask in
[ernilambar/tap](https://github.com/ernilambar/homebrew-tap). Homebrew does not
load a third-party tap until it is trusted, so tap and trust it first:

```bash
brew tap ernilambar/tap
brew trust ernilambar/tap
brew install --cask termiz
```

The cask requires macOS on arm64 and installs `Termiz.app` into the usual
`/Applications` location. Its postflight removes the quarantine attribute, so
the ad hoc-signed bundle launches without a Gatekeeper detour. Updates arrive
through the tap:

```bash
brew upgrade --cask termiz
```

## Install a release manually

Pushing a `v*` tag publishes `<Name>-macos-arm64.zip` and `SHA256SUMS` on the
GitHub release. The bundle is ad hoc signed, so macOS quarantines it and
Finder refuses the first launch until the attribute is removed:

```bash
unzip <Name>-macos-arm64.zip
xattr -dr com.apple.quarantine <Name>.app
mv <Name>.app /Applications/
open /Applications/<Name>.app
```

Verify the archive first if you like:

```bash
shasum -a 256 -c SHA256SUMS
```

## Build from source

```bash
git clone git@github.com:ernilambar/termiz.git
cd termiz
go mod download
go build -o termiz .
./termiz
```

`./termiz` opens the native window with a live shell; it needs a graphical
macOS session. To produce the same archive a release publishes, run
`./scripts/bundle.sh` — it runs the quality gate, builds with the release
flags, and writes `dist/<Name>.app`, `dist/<Name>-macos-arm64.zip`, and
`dist/SHA256SUMS`.

## Install the app bundle

A bare binary has no bundle identity: no Dock icon, no app menu, and no
LaunchServices registration. `dev.sh` builds the `.app` with go-gui's
`buildapp`, installs it to `/Applications`, and relaunches it:

```bash
./scripts/dev.sh             # debug build
./scripts/dev.sh --release   # release flags, verifies the arm64 binary
```

The installed copy is replaced and the running one is killed first, so any
shell session in it is lost. Use the scripts when testing bundle-only behavior;
use `go build -o termiz .` for a quick loop.

## The `termiz` command

`dev.sh` installs a `termiz` CLI shim in `/usr/local/bin` when that directory
is writable, otherwise in `~/.local/bin`. Set `TERMIZ_BIN_DIR` to choose
another directory; the script prints a `PATH` hint when the chosen directory
is not on it. With no arguments the shim launches through `open(1)`, so
LaunchServices supplies the Dock icon, activation, and single-instance
behavior, and it forwards `VISUAL` and `EDITOR` so `Termiz > Settings…` uses
the editor your shell would. Any argument is passed to the bundle binary
directly, so informational commands print to the terminal.

The informational commands are:

```bash
termiz --help      # or -h, or `help`
termiz --version   # or -V, or `version`
termiz themes      # list the bundled color themes
```

An unknown argument prints the usage to stderr and exits with status 2 without
opening a window.

## Uninstall

When installed with Homebrew:

```bash
brew uninstall --cask termiz
brew untap ernilambar/tap   # once no other cask from the tap is installed
```

A manual install removes the bundle and the CLI shim directly:

```bash
rm -rf "/Applications/<Name>.app"
rm -f /usr/local/bin/termiz ~/.local/bin/termiz   # whichever dev.sh installed
```

User data is separate from the bundle:

- `~/.config/termiz/config.toml`, or the file named by `config_file`, is left
  untouched by the app. Delete it to drop your settings.
- `~/.local/state/termiz/state.toml` holds the last shell directory.
- `~/Library/Logs/termiz.log` holds the app log.
