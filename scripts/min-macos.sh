#!/usr/bin/env bash
# Set the bundle's minimum macOS version to 15.0 and re-sign it.
#
# go-gui's buildapp writes LSMinimumSystemVersion 11.0 unconditionally and has
# no flag for it, so the bundle has to be corrected after packaging. Editing
# Info.plist invalidates buildapp's signature, so the bundle is signed again
# with the same identity buildapp used.
set -euo pipefail

if [ "$#" -ne 1 ]; then
    printf 'usage: %s <App.app>\n' "$(basename "$0")" >&2
    exit 2
fi

app="$1"
plist="$app/Contents/Info.plist"
min_macos="15.0"

if [ ! -f "$plist" ]; then
    printf 'error: %s not found\n' "$plist" >&2
    exit 1
fi

/usr/libexec/PlistBuddy -c "Set :LSMinimumSystemVersion $min_macos" "$plist" 2>/dev/null ||
    /usr/libexec/PlistBuddy -c "Add :LSMinimumSystemVersion string $min_macos" "$plist"

# buildapp's default is ad hoc; BUILDAPP_SIGN_IDENTITY is its documented way to
# supply a real identity, so honour the same variable here.
codesign --force --sign "${BUILDAPP_SIGN_IDENTITY:--}" "$app"
