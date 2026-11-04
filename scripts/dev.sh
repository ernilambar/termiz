#!/usr/bin/env bash
# Build the app bundle, install it to /Applications and relaunch it.
#
#   ./scripts/dev.sh             debug build
#   ./scripts/dev.sh --release   release build, same flags as the release workflow
#
# Installing the bundle is what makes bundle-only behaviour testable: the Dock
# icon, the app menu and LaunchServices registration come from the bundle, not
# from a bare binary. The copy already in /Applications is killed first, so its
# running shell is lost.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

# The display name lives in appinfo.toml alone; reading it here keeps a rename
# to one file.
app_name="$(sed -n 's/^name[[:space:]]*=[[:space:]]*"\(.*\)"$/\1/p' appinfo.toml)"
if [ -z "$app_name" ]; then
    printf 'error: appinfo.toml has no name\n' >&2
    exit 1
fi

dest="/Applications/$app_name.app"
app="$root/dist/$app_name.app"
lsregister="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

usage() {
    cat <<'EOF'
usage: scripts/dev.sh [--release]

  --release  build with the release workflow flags and verify the arm64 arch

Also installs a `termiz` CLI shim in /usr/local/bin when writable, else in
~/.local/bin. Set TERMIZ_BIN_DIR to choose the directory.
EOF
}

release=false
for arg in "$@"; do
    case "$arg" in
        --release) release=true ;;
        -h | --help)
            usage
            exit 0
            ;;
        *)
            printf 'unknown argument: %s\n\n' "$arg" >&2
            usage >&2
            exit 2
            ;;
    esac
done

if [ "$(uname -s)" != "Darwin" ]; then
    printf 'error: this app targets macOS only (this host is %s)\n' "$(uname -s)" >&2
    exit 1
fi

# hw.optional.arm64 stays 1 on Apple silicon even under Rosetta, where uname -m
# reports x86_64; checking the hardware keeps an Intel build from being tested
# as if it were a supported target.
if [ "$(sysctl -n hw.optional.arm64 2>/dev/null || echo 0)" != "1" ]; then
    printf 'error: Apple silicon required; this is an Intel Mac\n' >&2
    exit 1
fi

# The version lives in main.go; the release tag must match it.
version="$(sed -n 's/^const version = "\(.*\)"$/\1/p' main.go)"
if [ -z "$version" ]; then
    printf 'error: main.go has no version constant\n' >&2
    exit 1
fi

mkdir -p "$root/dist"

profile="debug"
if $release; then
    profile="release"
fi
printf '==> go build (%s)\n' "$profile"
if $release; then
    CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
        go build -trimpath -ldflags="-s -w" -o dist/termiz .
    arch="$(lipo -archs dist/termiz)"
    if [ "$arch" != "arm64" ]; then
        printf 'error: expected arm64 binary, got: %s\n' "$arch" >&2
        exit 1
    fi
else
    go build -o dist/termiz .
fi

printf '==> buildapp\n'
# go run pins buildapp to the go-gui version in go.mod. The manifest supplies
# the bundle ID and display name; -version wins over the manifest, which has
# no version, so a release tag is not duplicated in a file.
go run github.com/go-gui-org/go-gui/cmd/buildapp \
    -o "$root/dist" \
    -manifest "$root/appinfo.toml" \
    -version "$version" \
    "$root/dist/termiz"

if pkill -f "$dest/Contents/MacOS/"; then
    while pgrep -f "$dest/Contents/MacOS/" >/dev/null; do
        sleep 0.1
    done
fi

rm -rf "$dest"
ditto "$app" "$dest"
# Finder only notices a replaced bundle once LaunchServices forgets the old one.
"$lsregister" -f "$dest"

printf 'Installed %s (%s) to %s\n' "$app" "$version" "$dest"

# The CLI is a shim, not a symlink to the app's main binary: a bare `termiz`
# must go through open(1) so LaunchServices gives the app its Dock icon,
# activation and single-instance behavior, while informational commands exec
# the bundle binary directly so their output reaches the terminal. The usual
# /usr/local/bin is on the default PATH but root-owned on stock macOS, so fall
# back to a per-user directory rather than requiring sudo.
bin_dir="${TERMIZ_BIN_DIR:-}"
if [ -z "$bin_dir" ]; then
    if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
        bin_dir=/usr/local/bin
    else
        bin_dir="$HOME/.local/bin"
    fi
fi
mkdir -p "$bin_dir"
cli="$bin_dir/termiz"
cat >"$cli" <<EOF
#!/bin/sh
# Installed by termiz's scripts/dev.sh.
app="$dest"
if [ "\$#" -eq 0 ]; then
    # open(1) launches through LaunchServices, which does not inherit this
    # shell's environment; forward the editor variables so Termiz > Open
    # Config File… uses the editor this shell would.
    set -- "\$app"
    if [ -n "\$VISUAL" ]; then set -- --env "VISUAL=\$VISUAL" "\$@"; fi
    if [ -n "\$EDITOR" ]; then set -- --env "EDITOR=\$EDITOR" "\$@"; fi
    exec open "\$@"
fi
exec "\$app/Contents/MacOS/termiz" "\$@"
EOF
chmod +x "$cli"

printf 'Installed CLI %s\n' "$cli"
case ":$PATH:" in
    *":$bin_dir:"*) ;;
    *)
        printf 'note: %s is not on PATH; add it to run termiz directly:\n' "$bin_dir"
        printf '  export PATH="%s:$PATH"\n' "$bin_dir"
        ;;
esac

open "$dest"
