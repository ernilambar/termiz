#!/usr/bin/env bash
# Rebuild third_party/go-term from an upstream tag plus the local patches.
#
#   ./scripts/update-goterm.sh          re-apply the patches onto the recorded tag
#   ./scripts/update-goterm.sh v0.18.0  move to a new upstream tag
#
# termiz builds against a patched copy of go-term (go.mod replaces the module
# with ./third_party/go-term). The copy is generated: never edit it directly.
# Change go-term by adding or editing a file in third_party/go-term/patches
# (git format-patch output) and running this script. A patch that no longer
# applies stops the script with the clone left behind, so it can be fixed with
# `git am` there and re-exported.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dest="$root/third_party/go-term"
upstream="https://github.com/go-gui-org/go-term.git"

tag="${1:-$(cat "$dest/UPSTREAM")}"
if [ -z "$tag" ]; then
    printf 'error: no tag given and %s/UPSTREAM is empty\n' "$dest" >&2
    exit 1
fi

work="$(mktemp -d)"
git -c advice.detachedHead=false clone --quiet --depth 1 --branch "$tag" "$upstream" "$work"
# git am needs an identity to create the commits; it never leaves the clone.
if ! git -C "$work" -c user.name=termiz -c user.email=termiz@localhost \
    am --quiet --3way "$dest"/patches/*.patch; then
    printf 'error: a patch failed to apply on %s; resolve it in %s\n' "$tag" "$work" >&2
    exit 1
fi

# Only what termiz builds against, the licence the MIT terms require, and the
# shell-integration scripts go-term's own tests read.
find "$dest" -mindepth 1 -maxdepth 1 ! -name patches -exec rm -rf {} +
for f in go.mod go.sum LICENSE term internal; do
    cp -R "$work/$f" "$dest/"
done
mkdir -p "$dest/scripts"
cp -R "$work/scripts/shell-integration" "$dest/scripts/"
printf '%s\n' "$tag" >"$dest/UPSTREAM"
rm -rf "$work"

printf 'third_party/go-term: %s + %d patches\n' "$tag" "$(ls "$dest"/patches/*.patch | wc -l | tr -d ' ')"
