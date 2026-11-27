# copiar

A dual-pane terminal file manager with Vim keys: browse two folders side by side and press <kbd>space</kbd> to copy from one to the other — in either direction.

## Quick path

1. Install — download the archive for your system from the [Releases](https://github.com/RHAM91/copy-tui/releases) page and put `copiar` (`copiar.exe` on Windows) in your `PATH`, or build it with Go 1.24+:

   | System | Archive |
   |---|---|
   | Linux (x86-64 / ARM64) | `copiar_<version>_linux_amd64.tar.gz` / `_linux_arm64.tar.gz` |
   | macOS Intel / Apple Silicon | `copiar_<version>_darwin_amd64.tar.gz` / `_darwin_arm64.tar.gz` |
   | Windows (x86-64 / ARM64) | `copiar_<version>_windows_amd64.zip` / `_windows_arm64.zip` |

   ```bash
   go install github.com/RHAM91/copy-tui/cmd/copiar@latest
   ```

   > **macOS:** binaries are not signed, so the first run of a file downloaded with a browser is blocked by Gatekeeper. Allow it once with `xattr -d com.apple.quarantine ./copiar`. Updates through `copiar update` are not affected.

2. Run it in any folder — both panels open there:

   ```bash
   copiar            # with Nerd Font icons
   copiar --no-icons # if your terminal font has no Nerd Font glyphs
   copiar --version  # print the installed version
   copiar update     # install the latest release
   ```

3. Move to the destination in one panel, <kbd>tab</kbd> to the other, select a file and press <kbd>space</kbd>.

Press <kbd>?</kbd> inside the app for the full keymap.

## Keys

| Navigation | | Files | |
|---|---|---|---|
| `j` / `k` | down / up | `space` | copy to the other panel |
| `h` / `l` | parent / open folder | `x` | move to the other panel (asks first) |
| `gg` / `G` | first / last entry | `d` | move to trash (asks first) |
| `ctrl+d` / `ctrl+u` | half page down / up | `D` | delete permanently (asks first, defaults to Cancel) |
| `tab` | switch panel | `r` | rename |
| `/`, `n` / `N` | search, next / previous match | `a` | new folder |
| `.` | show / hide hidden files | `?` | toggle help |
| `ctrl+s` | save both panel paths as a bookmark | `q` | quit |
| `ctrl+b` | open the bookmark table | | |
| `=` | same folder in both panels | | |
| `ctrl+t` | choose a theme | | |

**Selection:** `m` marks the entry and moves down, `v` selects a range (press `v` again to keep it), `esc` clears. Copy, move and delete apply to the selection, or to the entry under the cursor when nothing is selected.

## How operations behave

| Topic | Behavior |
|---|---|
| Copy semantics | Like `cp -a`: folders are copied recursively, symlinks are copied as links (never followed), permissions and modification times are kept. |
| Name conflicts | A dialog offers **Overwrite**, **Skip** or **Rename** (`name (1).ext`), with an *Apply to all* toggle. Overwriting a folder merges its contents. |
| Progress & cancel | Long operations show a progress modal; `esc` cancels. Files are written to a temporary name first, so a cancelled copy never leaves half-written files or destroys the file it was replacing. |
| Layout | The panels sit side by side; when the terminal is narrower than 100 columns they stack, the left panel on top and the right one below. |
| Themes | `ctrl+t` lists **Catppuccin Mocha** (default), **Kanagawa Wave**, **Kanagawa Dragon** and **Tokyo Night**. Moving through the list previews each theme; `enter` keeps and saves it, `esc` goes back to the previous one. Themes color text, borders and highlights and keep your terminal's background. The choice is saved as `theme = "kanagawa-wave"` in `config.toml`, next to the bookmarks (`~/.config/copiar/config.toml` on Linux). An unknown theme or an unreadable file falls back to the default with a message, and a corrupt file is never overwritten. |
| Same folder | Copying into the folder the item already lives in creates a duplicate `name (1).ext`. |
| Move | Renames within a file system; across file systems it copies and then deletes the source, so a cancelled move keeps the source. |
| Updates | Set `COPIAR_TUI_UPDATE_TOKEN` to your download token to enable them. When a newer release exists, the status bar shows `vX.Y.Z available — run copiar update` (checked in the background at startup; nothing is shown offline or without a token). `copiar update` downloads the release for your platform from the update catalog, verifies its size, SHA-256 and Ed25519 signature against the public key built into the binary, and replaces the binary; use `sudo copiar update` (or an Administrator terminal on Windows) if it lives in a system folder. Builds from source (`dev`) do not update. |
| Same folder | `=` asks which way to go: **Left → Right** opens the left folder in the right panel, **Right → Left** the opposite. The active panel's folder is preselected, focus stays where it was, and the panel that changes puts its cursor on the same entry. When both panels already show the same folder, only a message is shown. |
| Bookmarks | `ctrl+s` asks for a name and saves the left and right panel folders; an existing name asks before replacing it. `ctrl+b` opens the table: `enter` opens both folders, `r` renames and `d` deletes (both ask first). Before opening, both folders are checked: if one is gone you can **Go anyway** (only the existing one opens), **Delete** the broken bookmark or cancel; if both are gone you can delete it or cancel. Bookmarks are stored in `~/.config/copiar/bookmarks.json` on Linux, `~/Library/Application Support/copiar/` on macOS and `%AppData%\copiar\` on Windows. |
| Trash | `d` uses the system trash so items can be restored: the freedesktop.org trash on Linux (`~/.local/share/Trash`), Finder on macOS (with **Put Back**; falls back to `~/.Trash` when Finder cannot be used, e.g. over SSH), and the Recycle Bin on Windows. |

> **WSL note:** the trash is the Linux one. Items trashed from `/mnt/c/...` do not appear in the Windows Recycle Bin.

> **tmux note:** `ctrl+b` is tmux's default prefix, so tmux keeps it. Press it twice (`ctrl+b ctrl+b`) to send it to copiar, or change the prefix (for example `set -g prefix C-a`).

## Development

```bash
go test -race ./...          # unit and UI tests
go run ./cmd/copiar          # run from source
```

| Path | Responsibility |
|---|---|
| `cmd/copiar` | flags and program start |
| `internal/fsops` | file operations (list, copy, move, trash, delete, rename), no UI; OS-specific code in `*_linux.go`, `*_darwin.go`, `*_windows.go`, `*_unix.go` |
| `internal/bookmarks` | named left/right folder pairs saved as JSON in the user config directory |
| `internal/config` | preferences (the theme) in `config.toml` in the user config directory |
| `internal/updater` | update catalog client: version check, download, SHA-256 and Ed25519 verification |
| `internal/update` | self-update service on top of the catalog client |
| `internal/ui` | Bubble Tea model: panels, dialogs, keymap, theme (Catppuccin Mocha) |

## Releases

Releases are published to the private update catalog by `.github/workflows/release.yml` when a version tag is pushed:

```bash
git tag v0.6.0 && git push origin v0.6.0
```

The workflow runs `go test -race`, builds the four enabled targets (`linux/amd64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`), signs each binary and publishes them together; the catalog only exposes a version once all targets are uploaded. The tag must look like `vMAJOR.MINOR.PATCH`, and the catalog rejects an already published or lower version. The `TARGETS` list in the workflow must match the targets checked in the catalog panel.

Required repository secrets: `CATALOG_URL`, `CATALOG_UPLOAD_TOKEN`, `CATALOG_SIGNING_KEY`.
