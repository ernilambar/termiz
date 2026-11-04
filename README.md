# terming

A terminal emulator built on [go-gui](https://github.com/go-gui-org/go-gui) and
[go-term](https://github.com/go-gui-org/go-term).

One window, one live shell on a PTY, rendered through go-gui's GPU backend.
The shell is `$SHELL`; when the environment carries no `SHELL` (a Finder launch),
the account's login shell is resolved instead of falling back to `/bin/sh`.
Closing the window tears down the shell, and the window closes when the shell
exits.

## Build and run

Requires Go 1.26+ and, on macOS, a C toolchain (the Metal backend is
Objective-C). Linux and Windows build cgo-free.

```bash
go build -o terming .
./terming
```

Configuration and theme selection are a later phase; this is the live-terminal
foundation.
