//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

// termizZoomWindow zooms the app's window to the screen's visible area, the
// same operation as the Window > Zoom menu item. Key window first: the window
// go-gui just made key is the one to grow. mainWindow and the first visible
// window cover a launch where AppKit has not keyed it yet.
static void termizZoomWindow(void) {
	NSWindow *win = [NSApp keyWindow];
	if (win == nil) {
		win = [NSApp mainWindow];
	}
	if (win == nil) {
		for (NSWindow *candidate in [NSApp windows]) {
			if ([candidate isVisible]) {
				win = candidate;
				break;
			}
		}
	}
	// zoom: toggles, so an already-zoomed window would shrink back.
	if (win != nil && ![win isZoomed]) {
		[win zoom:nil];
	}
}
*/
import "C"

// maximizeWindow fills the screen's visible area. go-gui has no maximize or
// fullscreen call, and the AppKit window is only reachable through cgo;
// OnInit runs on the main thread with the window already created and key, so
// the zoom happens before any terminal content is painted.
//
// Zoom, not fullscreen: the title bar and traffic lights stay, and the window
// stays on the current Space, which is what a terminal wants from "maximized".
func maximizeWindow() { C.termizZoomWindow() }
