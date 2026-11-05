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
```

## Keys

### Top level

| Key | Type | Default | Meaning |
| --- | --- | --- | --- |
| `config_file` | string | `""` | Another file to load as a base before this one; see [Includes](#includes). |

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

`maximize` is macOS-only. A Linux or Windows build, or a cgo-free Darwin
build, has no AppKit window to zoom and keeps the configured size.

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

`cursor_style` is matched case-insensitively, and `beam` is accepted as a
synonym for `bar`; an empty value keeps `block`, while any other spelling is
logged and replaced by `block`. The three keys describe the cursor the window
starts with and the one `reset` (the shell command and the escape sequences)
returns to. They are defaults, not overrides: with `cursor_lock = false` an
application can still change the shape, as vim does for insert mode. Setting
`cursor_lock = true` makes the window drop those requests, so the configured
shape and blink hold for the whole session.

`shell` is taken literally as the program to run; use an absolute path to be
unambiguous. A configured shell outranks `$SHELL`. When the chosen shell
differs from the inherited `$SHELL`, the child is given the matching `SHELL`,
so subshells and tools that re-exec `$SHELL` agree with the window.

`dir` expands a leading `~` and `$VAR` references. The value must name an
existing directory at load time; otherwise it is logged and treated as empty,
falling through to the previous session's directory and then home.

## Precedence

For `shell`:

1. `terminal.shell`, when non-empty.
2. `$SHELL`, when set.
3. The OS login shell (`dscl` on macOS, `getent` on Linux).
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

Loading never fails the launch. Problems are written to the log and the
affected value is replaced by its default:

| Problem | Result |
| --- | --- |
| Config file missing | All defaults. |
| File does not parse | The whole file is ignored. |
| Unknown key | Logged; the key is ignored; other keys still apply. |
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

`Termiz > Settings` (⌘,) opens the config in `$VISUAL`, then `$EDITOR`, then
the system text editor. It creates the config directory when missing and seeds
a missing file with a comment pointing at this reference page — the system
editor also needs the path to exist. It never changes any value.

## Verifying a config

The loader logs under the `config:` prefix. To check a file without a GUI:

```bash
go test ./... -run Config
```

Or launch `./termiz` and inspect the log; every warning names the file it came
from, except repairs.
