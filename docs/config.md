# Configuration

termiz reads one optional top-level TOML file at startup. Every key is
optional: a key that is absent keeps its built-in default, so an empty or
missing file is valid.

```
$XDG_CONFIG_HOME/termiz/config.toml
```

When `XDG_CONFIG_HOME` is unset, or is not an absolute path, the file is
`~/.config/termiz/config.toml`. A relative `XDG_CONFIG_HOME` is ignored on
purpose: a GUI launch resolves it against the working directory, which is `/`
for a Finder launch.

The file is read once, before the window opens; a change applies on the next
launch. termiz never changes its contents — editing it is the only way to
change settings.

## Example

```toml
# Load another file first; this file's own keys still win.
config_file = "~/dotfiles/termiz/config.toml"

[window]
maximize   = true   # macOS: fill the visible screen area on launch
width      = 1000
height     = 640
min_width  = 400
min_height = 240
padding    = 8       # 0–64; inset from the window edge to the terminal

[font]
family = "Menlo"
size   = 12          # 6–72

[theme]
name = "Catppuccin Mocha"  # any bundled theme; empty: go-term's default

[terminal]
scrollback_rows = 10000  # 0: go-term's 5000 rows; negative: no scrollback
cursor_style    = "block"  # block, underline or bar
cursor_blink    = true
cursor_lock     = false    # ignore a child's DECSCUSR cursor changes
shell           = ""     # empty: $SHELL, then the OS login shell
dir             = ""     # empty: last session's directory, then home
allow_osc52     = false  # let programs write the clipboard (OSC 52)
download_dir    = ""     # OSC 1337 File= transfers; empty: disabled

[keybindings]
copy           = "Cmd+Shift+C"
font-inc       = "Cmd+="
scroll-page-up = "none"  # hand PageUp back to the child
"copy-mode.down" = "Ctrl+N"
```

## Keys

### Top level

| Key | Type | Default | Meaning |
| --- | --- | --- | --- |
| `config_file` | string | `""` | Another file to load as a base before this one; see [Includes](#includes). |
| `[keybindings]` | table | — | Terminal key overrides; see [`[keybindings]`](#keybindings). |

### `[window]`

| Key | Type | Default | Meaning |
| --- | --- | --- | --- |
| `maximize` | bool | `true` | Open filling the screen's visible area, like Window > Zoom. Set `false` to open at `width`×`height`. |
| `width` | int | `1000` | Initial window width in points; must be positive. |
| `height` | int | `640` | Initial window height in points; must be positive. |
| `min_width` | int | `400` | Minimum width the user can resize to; must be positive. |
| `min_height` | int | `240` | Minimum height the user can resize to; must be positive. |
| `padding` | int | `8` | Inset in points from the window edge to the terminal; `0`–`64`. |

`padding` gives the grid breathing room and keeps the first column clear of
the window's rounded corners. `0` is allowed and leaves the grid at the
widget's own 4 px inset; the text therefore starts `padding + 4` points from
each edge. A negative or larger-than-`64` value is logged and replaced by `8`.

A `min_width`/`min_height` larger than the opening size is lowered to
`width`/`height` rather than forcing the window to start below its own floor.

### `[font]`

| Key | Type | Default | Meaning |
| --- | --- | --- | --- |
| `family` | string | `"Menlo"` | Font family name. An empty value quietly falls back to the default. |
| `size` | number | `12` | Point size, from 6 through 72 inclusive. |

The size range is a usability bound: below 6 points text is unreadable, above
72 a single cell no longer fits the 24×80 grid the PTY starts with. A value
outside the range is logged and replaced by 12.

### `[theme]`

| Key | Type | Default | Meaning |
| --- | --- | --- | --- |
| `name` | string | `""` | A theme from go-term's bundled iTerm2-Color-Schemes corpus, or `Default`. |

Matching is case-insensitive, so `catppuccin mocha` and `Catppuccin Mocha`
select the same theme; the corpus spelling is what gets stored internally.
Empty and `Default` both select go-term's built-in dark theme. The window
chrome (title bar, focus ring) follows the theme's light or dark character.

`termiz themes` prints every valid name, with `Default` first:

```bash
termiz themes | grep -i catppuccin
```

An unknown name is logged and the built-in default is used. There is no
partial palette fallback.

### `[terminal]`

| Key | Type | Default | Meaning |
| --- | --- | --- | --- |
| `scrollback_rows` | int | `10000` | History rows kept above the screen. `0` keeps go-term's own default (5000), a positive value caps history, a negative value disables it. |
| `cursor_style` | string | `"block"` | Cursor shape: `block`, `underline` or `bar`. |
| `cursor_blink` | bool | `true` | Whether the cursor blinks. |
| `cursor_lock` | bool | `false` | Ignore the child's cursor-shape requests (DECSCUSR) for the life of the window. |
| `shell` | string | `""` | Shell to run. Empty selects `$SHELL`, then the OS login shell. |
| `dir` | string | `""` | Starting directory. Empty restores the previous session's directory, then home. |
| `allow_osc52` | bool | `false` | Let a program write the system clipboard through OSC 52. |
| `download_dir` | string | `""` | Directory for OSC 1337 `File=` transfers. Empty disables transfers. |

`cursor_style` is matched case-insensitively, and `beam` is accepted as a
synonym for `bar`; an empty value keeps `block`, while any other spelling is
logged and replaced by `block`. The three keys describe the cursor the window
starts with and the one `reset` (the shell command and the escape sequences)
returns to. They are defaults, not overrides: with `cursor_lock = false` an
application can still change the shape, as vim does for insert mode. Setting
`cursor_lock = true` makes the window drop those requests, so the configured
shape and blink hold for the whole session.

`shell` is the program to run. `~` and `$VAR` expand, and a bare name is
looked up in `PATH`, then — because a Dock or Finder launch has launchd's
minimal `PATH` — in the `PATH` your login shell builds. An absolute path skips
both lookups. A shell that cannot be found is
logged and the login-shell resolution below applies instead. A known shell
(sh, bash, zsh, fish, dash, ksh, mksh, tcsh, csh, nu, xonsh) starts as a login
shell (`-l`), as Terminal.app does, so `~/.zprofile` runs; any other shell is
started without the flag, since one that rejects it would exit at once. A configured shell outranks `$SHELL`. When the chosen shell
differs from the inherited `$SHELL`, the child is given the matching `SHELL`,
so subshells and tools that re-exec `$SHELL` agree with the window.

`dir` expands a leading `~` and `$VAR` references, and a relative path is
relative to the home directory. The value must name an
existing directory at load time; otherwise it is logged and treated as empty,
falling through to the previous session's directory and then home.

`allow_osc52` lets a program write the system clipboard with OSC 52. It is
off by default because any output in the pane — a hostile banner, a script you
ran, a remote login — could otherwise replace the clipboard silently, and a
planted payload runs with your privileges at the next paste. Turn it on only
for programs you trust.

`download_dir` enables OSC 1337 `File=` transfers (iTerm2's `imgcat -d`,
`it2dl`) and names where they are saved. Empty, the default, disables them:
terminal output must not create files on its own authority. Files land with
0600 permissions and never overwrite an existing name — a collision gets a
`" (N)"` suffix. The directory is created on first use; a path that exists but
is not a directory is logged and disables transfers. `~` and `$VAR` expand,
and a relative path resolves against the home directory.

### `[keybindings]`

Each entry rebinds one of go-term's terminal actions. A name may be bare
(`copy`), namespaced (`term.copy`), or written as a nested table
(`[keybindings.term]`); all three spell the same action. Quote a dotted name
that is not meant as a table: `"copy-mode.down" = "Ctrl+N"`.

A chord is modifiers first, then the key, joined with `+`. Modifiers are
`Cmd` (alias `Super`), `Ctrl`, `Alt` (alias `Opt`) and `Shift`; matching is
case-insensitive. Key names are `A`–`Z`, `0`–`9`, `F1`–`F25`, `Space`,
`Enter`/`Return`, `Escape`/`Esc`, `Tab`, `Backspace`, `Delete`/`Del`,
`Insert`, `Home`, `End`, `PageUp`, `PageDown`, `Left`, `Right`, `Up`, `Down`,
and the punctuation keys `[` `]` `\` `/` `;` `,` `.` `-` `=` `` ` `` (also
spelled `LeftBracket`, `Backslash`, `Slash`, `Semicolon`, `Comma`, `Period`,
`Minus`, `Equal`, `GraveAccent`).

```toml
[keybindings]
copy           = "Cmd+Shift+C"
find           = "Cmd+F"
scroll-page-up = "none"   # hand PageUp back to the child
"copy-mode.down" = "Ctrl+N"
```

The terminal actions are `copy`, `paste`, `find`, `toggle-regex`,
`next-match`, `prev-match`, `prev-prompt`, `next-prompt`, `jump-failure`,
`select-output`, `select-all`, `scroll-page-up`, `scroll-page-down`,
`scroll-top`, `scroll-bottom`, `font-inc`, `font-dec`, `font-reset`,
`copy-mode`, `hints`, `hints-copy`, and the `copy-mode.*` navigation actions
(`copy-mode.down`, `copy-mode.yank`, …).

`none` — or an empty value — unbinds the action, which hands the key back to
the shell; this is how you free `Ctrl+R` for a shell's reverse-i-search. An
unknown action, an unparseable chord, and a second entry claiming a chord
already taken by an earlier entry are logged and ignored. An override
replaces the action's whole built-in chord list rather than adding to it, and
only terminal actions are rebindable — window commands such as Quit and
Settings belong to the native menu.

## Precedence

For `shell`:

1. `terminal.shell`, when non-empty.
2. `$SHELL`, when set.
3. The OS login shell (`dscl`).
4. go-term's fallback, only if the OS answer is also empty.

termiz never substitutes `/bin/sh` for an unresolved shell.

For `dir`:

1. `terminal.dir`, when it names an existing directory.
2. The directory the previous session ended in, if the saved path still
   exists.
3. The home directory.

## Includes

`config_file` names another file to load as the base for the current one. This
keeps the real settings in a dotfiles repository while
`~/.config/termiz/config.toml` stays a one-line pointer:

```toml
# ~/.config/termiz/config.toml
config_file = "~/dotfiles/termiz/config.toml"
```

Resolution rules:

- The named file is loaded first, then the file carrying the key is decoded
  again over it, so the file holding `config_file` wins for every value it
  sets itself.
- A relative path resolves against the directory of the file that names it,
  not the working directory.
- `~` and `$VAR` expand in the path.
- A missing target deeper in the chain is logged and skipped; the rest of the
  file still applies. A missing top-level config file is normal and silent.
- A cycle, or a chain more than 8 includes deep, is logged and skipped rather
  than followed forever.
- A file that does not parse is ignored whole; nothing from it is applied.

## Error handling

Loading never fails the launch. Problems are written to the log —
`~/Library/Logs/termiz.log`, which Console.app also shows, and stderr when run
from a terminal — and the affected value is replaced by its default:

| Problem | Result |
| --- | --- |
| Config file missing | All defaults. |
| File does not parse (syntax error) | The whole file is ignored. |
| Unknown key | Logged; the key is ignored; other keys still apply. |
| Wrong type (`width = "1000"`) | Logged; that value reverts to its default; other keys still apply. |
| Out-of-range value | Logged; that value reverts to its default. |

Repairs are reported once, after all files are merged, and do not name a
single file because they describe the merged result.

## Related files and commands

The app never modifies a config value. The directory the shell ends in is
remembered separately, in
`$XDG_STATE_HOME/termiz/state.toml` (default
`~/.local/state/termiz/state.toml`), written atomically with mode `0600`.
While the session runs the directory is sampled every 2 seconds and written
only when it changes; it is written once more at close, so a shell that exits
on its own still leaves the last directory. Naming `terminal.dir` pins the
start directory and stops the state write entirely.

`Termiz > Settings…` (⌘,) opens the config in `$VISUAL`, then `$EDITOR`, then
the system text editor. The editor command is looked up in the `PATH` the
terminal's shell builds as an interactive login shell (so `~/.zshrc` counts),
because a Dock launch's own `PATH` misses Homebrew and editor CLIs. Quoting is
honored, so `"/Applications/Sublime Text.app/…/subl" -w` works; a command that
is still not found, or a terminal editor (vim, nvim, nano, emacs,
…), falls back to the system text editor. It creates the config directory when
missing and seeds
a missing file with a comment pointing at this reference page — the system
editor also needs the path to exist. It never changes any value.

## Verifying a config

The loader logs under the `config:` prefix. To check a file without a GUI:

```bash
go test ./... -run Config
```

Or launch termiz and read `~/Library/Logs/termiz.log`; every warning names the
file it came from, except repairs.
