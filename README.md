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
| `h` / `l` | parent / open folder | `x` | move to the other panel |
| `gg` / `G` | first / last entry | `d` | move to trash (asks first) |
| `ctrl+d` / `ctrl+u` | half page down / up | `D` | delete permanently (asks first, defaults to Cancel) |
| `tab` | switch panel | `r` | rename |
| `/`, `n` / `N` | search, next / previous match | `a` | new folder |
| `.` | show / hide hidden files | `?` | toggle help |
| | | `q` | quit |

**Selection:** `m` marks the entry and moves down, `v` selects a range (press `v` again to keep it), `esc` clears. Copy, move and delete apply to the selection, or to the entry under the cursor when nothing is selected.

## How operations behave

| Topic | Behavior |
|---|---|
| Copy semantics | Like `cp -a`: folders are copied recursively, symlinks are copied as links (never followed), permissions and modification times are kept. |
| Name conflicts | A dialog offers **Overwrite**, **Skip** or **Rename** (`name (1).ext`), with an *Apply to all* toggle. Overwriting a folder merges its contents. |
| Progress & cancel | Long operations show a progress modal; `esc` cancels. Files are written to a temporary name first, so a cancelled copy never leaves half-written files or destroys the file it was replacing. |
| Same folder | Copying into the folder the item already lives in creates a duplicate `name (1).ext`. |
| Move | Renames within a file system; across file systems it copies and then deletes the source, so a cancelled move keeps the source. |
| Updates | When a newer release exists, the status bar shows `vX.Y.Z available — run copiar update` (checked in the background at startup; nothing is shown when offline). `copiar update` downloads the release for your platform, verifies it against `checksums.txt` and replaces the binary; use `sudo copiar update` (or an Administrator terminal on Windows) if it lives in a system folder. Builds from source (`dev`) do not update. |
| Trash | `d` uses the system trash so items can be restored: the freedesktop.org trash on Linux (`~/.local/share/Trash`), Finder on macOS (with **Put Back**; falls back to `~/.Trash` when Finder cannot be used, e.g. over SSH), and the Recycle Bin on Windows. |

> **WSL note:** the trash is the Linux one. Items trashed from `/mnt/c/...` do not appear in the Windows Recycle Bin.

## Development

```bash
go test -race ./...          # unit and UI tests
go run ./cmd/copiar          # run from source
```

| Path | Responsibility |
|---|---|
| `cmd/copiar` | flags and program start |
| `internal/fsops` | file operations (list, copy, move, trash, delete, rename), no UI; OS-specific code in `*_linux.go`, `*_darwin.go`, `*_windows.go`, `*_unix.go` |
| `internal/update` | release check and self-update (checksum verified) |
| `internal/ui` | Bubble Tea model: panels, dialogs, keymap, theme (Catppuccin Mocha) |

## Releases

Releases are automated from `main` with [release-please](https://github.com/googleapis/release-please) and [GoReleaser](https://goreleaser.com):

1. Merge work into `main` using [Conventional Commits](https://www.conventionalcommits.org) (`feat:` bumps the minor version, `fix:` the patch; `feat!:` or `BREAKING CHANGE:` the major after 1.0).
2. The **Release** workflow keeps a release PR open with the next version and `CHANGELOG.md`.
3. Merging that PR creates the `vX.Y.Z` tag and GitHub release; GoReleaser then attaches `copiar_<version>_<os>_<arch>.tar.gz` archives and `checksums.txt`.

Try the build locally without publishing: `goreleaser release --snapshot --clean`.
