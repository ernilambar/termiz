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
open "$dest"
