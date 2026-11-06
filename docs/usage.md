# Using termiz

## Launch

With no arguments, `termiz` opens one window running a real shell on a PTY. A
bundled launch (Finder, Dock, or the `termiz` CLI shim) goes through
LaunchServices; `./termiz` from a build directory runs the binary directly.
Informational commands — `help`, `version`, `themes` — print and exit without
opening a window.

The shell is chosen in this order: `terminal.shell` when set, then `$SHELL`,
then the OS login shell, and only then go-term's fallback. A known shell starts
as a login shell (`-l`) so `~/.zprofile` runs; termiz never substitutes
`/bin/sh`. See [Precedence](config.md#precedence) for the exact rules.

## Command line

```bash
termiz --help      # or -h, or `help`
termiz --version   # or -V, or `version`
termiz themes      # list the bundled color themes
```

`termiz themes` prints every valid `theme.name`, `Default` first, one per line,
so it pipes:

```bash
termiz themes | grep -i catppuccin
```

The printed version is `version` in `main.go`, the same value the release tag
must match and the bundle reports to children as `TERM_PROGRAM_VERSION`. An
unrecognized argument prints the usage to stderr and exits with status 2.

## Window and menus

termiz is one window with one shell: no tabs, panes, or session management.
Closing the window or quitting ends the shell. With
`window.confirm_on_quit = true`, a close or quit while the shell is still alive
asks first; a shell that exits on its own closes the window without a prompt.

The Termiz app menu carries:

- `Settings…` (⌘,) opens the config file, creating the directory and seeding a
  missing file with a comment that points at the configuration reference. The
  command is `$VISUAL`, then `$EDITOR`, then the system text editor; a terminal
  editor such as vim counts as unset because the app has no TTY. The editor is
  resolved with the terminal shell's `PATH`, so a Dock launch can find `code`
  or `subl`. The app never writes a setting.
- `Reload Configuration` (⌘⇧,) re-reads the config chain and applies the values
  that have live setters: theme, font, scrollback, cursor, keybindings, window
  padding, and quit confirmation. Window geometry, the shell, the start
  directory, OSC 52, and download transfers need a restart; the log names them.
- `Quit` (⌘Q).

The Window menu provides the usual Close, Minimize, and Zoom items. Window
maximization on launch is `window.maximize`, on by default.

## Session directory

With `terminal.dir` empty, the shell starts where the previous session ended.
termiz samples the shell's directory every 2 seconds and writes it to
`$XDG_STATE_HOME/termiz/state.toml` (default
`~/.local/state/termiz/state.toml`) only when it changes, plus once on close,
so an exited shell or a crash still leaves the last directory. The directory
comes from the shell's OSC 7 report when present — a report from a remote host
is ignored — and otherwise from the shell process, so no shell setup is
required. Naming `terminal.dir` pins the start directory and suppresses the
state write.

## Log

When a setting seems ignored or the window closes on launch, read
`~/Library/Logs/termiz.log` (also visible in Console.app). A GUI launch's
stderr goes nowhere, so the log receives the same lines. It is truncated at
launch once it passes 1 MiB. Configuration warnings carry a `config:` prefix
and name the value they replaced; changes that need a restart are called out.

## Next

- [Configuration](config.md) — every key, default, include rule, and error.
- [Installing](install.md) — releases, source builds, scripts, CLI shim.
