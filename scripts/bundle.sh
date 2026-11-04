#!/usr/bin/env bash
# Run the quality gate and build the publishable macOS bundle.
#
#   ./scripts/bundle.sh
#
# Produces the same artifacts .github/workflows/release.yml attaches to a
# release: the .app bundle, its -macos-arm64.zip archive and SHA256SUMS, all
# under dist/. Only darwin/arm64 is supported: no Intel build, no universal
# binary.
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

if [ "$(uname -s)" != "Darwin" ]; then
    printf 'error: this app targets macOS only (this host is %s)\n' "$(uname -s)" >&2
    exit 1
fi

# hw.optional.arm64 stays 1 on Apple silicon even under Rosetta, where uname -m
# reports x86_64; checking the hardware keeps an Intel build from passing as
# the supported target.
if [ "$(sysctl -n hw.optional.arm64 2>/dev/null || echo 0)" != "1" ]; then
    printf 'error: Apple silicon required; this is an Intel Mac\n' >&2
    exit 1
fi

# The tag is the one version source; appinfo.toml carries only the identity.
version="$(git describe --tags --abbrev=0 2>/dev/null || echo 0.0.0)"
version="${version#v}"

printf '==> gofmt\n'
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
    printf 'error: not gofmt-formatted:\n%s\n' "$unformatted" >&2
    exit 1
fi

printf '==> go vet\n'
go vet ./...

printf '==> go test\n'
go test ./...

printf '==> go build (release)\n'
rm -rf dist
mkdir -p dist
# Same flags as the release workflow so this is the artifact a tag publishes,
# not a debug build.
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
    go build -trimpath -ldflags="-s -w" -o dist/terming .

arch="$(lipo -archs dist/terming)"
if [ "$arch" != "arm64" ]; then
    printf 'error: expected arm64 binary, got: %s\n' "$arch" >&2
    exit 1
fi

printf '==> buildapp\n'
# go run pins buildapp to the go-gui version in go.mod. The manifest supplies
# the bundle ID and display name; -version wins over the manifest, which has
# no version, so a release tag is not duplicated in a file.
go run github.com/go-gui-org/go-gui/cmd/buildapp \
    -o "$root/dist" \
    -manifest "$root/appinfo.toml" \
    -version "$version" \
    "$root/dist/terming"

app_arch="$(lipo -archs "dist/$app_name.app/Contents/MacOS/terming")"
if [ "$app_arch" != "arm64" ]; then
    printf 'error: expected arm64 app binary, got: %s\n' "$app_arch" >&2
    exit 1
fi

printf '==> package\n'
zip="dist/$app_name-macos-arm64.zip"
# --norsrc --noextattr keeps AppleDouble ._* entries out of the archive; the
# signature lives in the Mach-O and CodeResources, not in an xattr.
ditto -c -k --norsrc --noextattr --keepParent "dist/$app_name.app" "$zip"
(cd dist && shasum -a 256 "$app_name-macos-arm64.zip" >SHA256SUMS)

printf '\nbundle:  %s\narchive: %s\ndigest:  %s\n' \
    "dist/$app_name.app" "$zip" "$(cat dist/SHA256SUMS)"
