package main

/*
#cgo CFLAGS: -mmacosx-version-min=15.0
#cgo LDFLAGS: -mmacosx-version-min=15.0
*/
import "C"

// termiz supports macOS 15 and later. The cgo flags above set the Mach-O
// LC_BUILD_VERSION minimum, which stops older systems from launching the app;
// scripts/min-macos.sh writes the matching LSMinimumSystemVersion into the
// bundle because go-gui's buildapp hardcodes 11.0.
