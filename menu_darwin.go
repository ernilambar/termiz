//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

// termizMoveMenuItems moves every item of the top-level menu named title into
// the application menu above Quit, adds a separator after them, then removes
// the emptied top-level menu. go-gui's app menu is built as About, separator,
// Quit and exposes no way to append an item, so moving an already-built item
// keeps go-gui's action target and representedObject wiring intact.
//
// Quit is identified as the app menu's last item, the layout go-gui builds;
// inserting above it keeps Quit last as convention expects.
static void termizMoveMenuItems(const char *title) {
	if (title == NULL) {
		return;
	}
	NSMenu *bar = [NSApp mainMenu];
	if (bar == nil) {
		return;
	}
	NSString *name = [NSString stringWithUTF8String:title];
	NSMenuItem *source = nil;
	for (NSMenuItem *top in bar.itemArray) {
		if (top.submenu != nil && [top.title isEqualToString:name]) {
			source = top;
			break;
		}
	}
	if (source == nil) {
		return;
	}
	NSMenuItem *appItem = bar.itemArray.firstObject;
	NSMenu *app = (appItem != nil) ? appItem.submenu : nil;
	if (app == nil) {
		return;
	}

	NSInteger index = app.numberOfItems - 1;
	if (index < 0) {
		index = 0;
	}
	// itemArray is a snapshot, so removing from the source menu while
	// iterating it is safe.
	for (NSMenuItem *item in source.submenu.itemArray) {
		[source.submenu removeItem:item];
		[app insertItem:item atIndex:index];
		index++;
	}
	[app insertItem:[NSMenuItem separatorItem] atIndex:index];
	[bar removeItem:source];
}
*/
import "C"

import "unsafe"

// moveMenuItemsIntoAppMenu moves the items of the top-level native menu named
// title into the app menu. go-gui has no API for app-defined app-menu items,
// and AppKit is only reachable through cgo; OnInit runs on the main thread
// with the menubar already installed.
func moveMenuItemsIntoAppMenu(title string) {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	C.termizMoveMenuItems(cTitle)
}
