# copiar-tui — dual-pane file manager TUI in Go

Objective: build `copiar`, a Vim-style dual-pane TUI to copy (and manage) files between folders on the same machine.
Branch: `feat/copiar-tui` (base `main` @ 701798f) · Delivery: `ask-on-risk` → `feature-branch-chain` · Test runner: `go test ./...` (stdlib `testing` + `teatest`).

## Specs

- **S1** Purpose: "Copiar archivos entre diferentes carpetas de la misma computadora, es como un cp pero con interface gráfica". Binary name: "el binarlo llamalo copiar". Module `github.com/RHAM91/copy-tui`, remote `https://github.com/RHAM91/copy-tui.git`.
- **S2** Layout: "debe tener pantalla dividida (doble planel), los paneles deben ubicarse en el directorio actual". Each panel navigates independently; `Tab` switches the active panel.
- **S3** Vim mode: "debe tener soporte modo vim". Confirmed keymap ("me sirve así"): `j`/`k` down/up, `h`/`l` parent/enter, `gg`/`G` top/bottom, `Ctrl+d`/`Ctrl+u` half page, `/` search with `n`/`N`, `Tab` switch panel, `m` mark, `v` visual selection, `Esc` clear selection, `space` copy, `.` toggle hidden, `?` full help, `q` quit.
- **S4** Copy: "al tener seleccionado un archivo si se presiona la tecla "espacio" se ejecuta el copiado entre carpetas, debe ser bidireccional". Copies from the active panel to the other one.
- **S5** Conflicts: "la 1, preguntar con diálogo" — overwrite / skip / rename (e.g. `archivo (1).txt`); with multiple items "vamos con el 1": each option has an "apply to all" variant.
- **S6** Directories: "la 1, recursivo con barra de progreso".
- **S7** Multi-select: "la 1, selección múltiple con v y m". `space` copies marked items, or the item under the cursor when nothing is marked.
- **S8** Scope: "vamos con el 3" — full manager: copy, move, delete, rename, mkdir. Keys ("me sirven"): `x` move to other panel, `r` rename (cursor item), `a` create folder in active panel.
- **S9** Delete: "la 3, ambos" — `d` sends to the freedesktop trash (`~/.local/share/Trash`), `D` deletes permanently; "d con confirmación también" — both confirm, `D` with a more explicit confirmation.
- **S10** Look: "la interface debe ser moderna, limpia, e intuitiva". "la 1, con flag --no-icons": Nerd Font icons + name + size, current path above each panel, active panel highlighted by border color; `--no-icons` disables icons. Palette: "la 1, catppuccin mocha".
- **S11** Help: "debe tener una barra con los comandos como ayuda". `?` shows full help.
- **S12** Long operations: "la 1, modal con Esc para cancelar" — modal progress dialog; `Esc` cancels and removes the partially copied file.
- **S13** Copy fidelity: "la 1, como cp -a" — symlinks copied as symlinks (not followed), permissions and mtime preserved.
- Assumption A1: entries sorted directories first, then case-insensitive by name; hidden files hidden by default.
- Assumption A2: errors (e.g. permission denied) are shown in the status bar, never crash the app.

## Tasks

| ID | Specs | Task | Route | Risk | Status | Commit |
|----|-------|------|-------|------|--------|--------|
| T1 | S1,S2,S10 | Scaffold: module, `cmd/copiar`, `--no-icons`, theme, two empty panels | inline | none | done | 66d9205 |
| T2 | S2,S3,S10,A1 | Panel navigation, listing, icons, size, path header, hidden toggle | inline | none | done | 1b944a5 |
| T3 | S3 | Search `/`, `n`, `N` | inline | none | done | a0f94e8 |
| T4 | S3,S7 | Selection `m`, `v`, `Esc` | inline | none | done | a0f94e8 |
| T5 | S5,S6,S12,S13 | Copy engine (fsops): recursive, cp -a, progress, cancel + cleanup, conflict naming | inline | item 1 | done | 1d9be95 |
| T6 | S4,S5,S7,S12 | Copy UI: `space`, conflict dialog with apply-to-all, progress modal | inline | item 1 | done | 5c701da |
| T7 | S8 | Move `x` (rename, cross-device fallback) | inline | item 1 | done | a7f5fb4 |
| T8 | S9 | Delete `d` trash / `D` permanent with confirmations | inline | item 1 | done | 646080e |
| T9 | S8 | Rename `r`, mkdir `a` | inline | none | done | 6b80665 |
| T10 | S11,A2 | Help bar, `?` help, status errors, README | inline | none | done | slice 1 |

Route evidence: no Writer trigger reason (tasks are sequential and depend on each other), so the parent writes inline. T5–T8 touch irreversible file effects (high-risk item 1) and get an independent verifier.

## Log

- **L1** (2026-11-17) Original request: "que tal hermano, quiero crear una tui en go, con el siguiente proposito: Copiar archivos entre diferentes carpetas de la misma computadora, es como un cp pero con interface gráfica, debe tener pantalla dividida (doble planel), los paneles deben ubicarse en el directorio actual, debe tener soporte modo vim, al tener seleccionado un archivo si se presiona la tecla "espacio" se ejecuta el copiado entre carpetas, debe ser bidireccional, la interface debe ser moderna, limpia, e intuitiva, debe tener una barra con los comandos como ayuda. Me puedría ayudar primero con la planificación, utiliza odd y hazme todas las preguntas necesarias para mejorar la implementación."
- **L2** (2026-11-17) Planning Q&A answers captured verbatim in S2–S13. Implementation authorized: "si, adelante". User note: tmux is available for interactive tests.
- **L3** (2026-11-17) Forecast ~2500 authored changed lines (> 400): `ask-on-risk` requires choosing a chain strategy before the first commit.
- **L4** (2026-11-17) Chain strategy: "vamos con el 1" → `feature-branch-chain` (slice PRs target `feat/copiar-tui`; one final merge to `main`). Empty root commit 701798f on `main`.
- **L5** T1 RED: `go test ./...` build failed (undefined `New`, `Options`, `parseArgs`). GREEN: `go vet ./...` clean; `go test ./...` ok (cmd/copiar, internal/ui).
- **L6** T2 RED: build failed (undefined `ListDir`, `Current`, `Status`). GREEN: `go test ./...` ok. Runtime: tmux 110x22 render shows icons (U+F07B), active border mauve, cursor background #313244 on active panel, #181825 on inactive.
- **L7** T3+T4 RED: build failed (undefined `Searching`, `SearchQuery`, `Selection`, `Visual`). One test fixed (wrong expected order: entries sort as one, three, two). GREEN: `go test ./...` ok. T3 and T4 share `app.go` key routing, so they ship in one commit.
- **L8** T5 RED: build failed (undefined `Copy`, `CopyOptions`, `UniqueName`, `chunkSize`). GREEN first run: `go test ./internal/fsops/` ok (13 tests: content/perm/mtime, recursive dirs, symlinks incl. loop and broken, progress totals, skip/overwrite/rename, next free name, dir merge, type mismatch, same-dir duplicate, copy-into-itself, cancel cleanup, cancel during overwrite keeps original, resolver error). Design: temp file + rename, so cancel never leaves partial files nor destroys the overwritten file. Verification: one independent verifier will cover T5, T7 and T8 together (all item 1, all in `internal/fsops`) to keep verification as rare as possible.
- **L9** T6 RED: build failed (undefined `Busy`, `ConflictPending`, `startOp`). GREEN: `go test -race -count=3 ./...` ok. Runtime (tmux 110x24): conflict dialog rendered over panels, `o` overwrote `dst/report.txt` ("new"), 300 MB copy completed. Icon rows looked shifted in `capture-pane` only (glyph display artifact; `--no-icons` capture aligned, Go-side widths equal). Dialog widened to 60 so hints fit one line.
- **L10** T7: refactored `Copy` conflict handling into `copier.target` (shared with `Move`); all 13 copy tests stayed green, proving prior behavior. `ErrCopyIntoItself` renamed `ErrIntoItself` (unreleased). RED: build failed (undefined `Move`, `rename`); UI tests failed (no conflict dialog on `x`). GREEN: `go test -race ./...` ok; cross-device path tested by injecting EXDEV through the `rename` variable (copy+delete, cancel keeps source and cleans destination).
- **L11** T8 RED: build failed (undefined `Trash`, `Remove`, `TrashDir`, `ConfirmPending`). GREEN: `go test -race ./...` ok. One UI test changed: `q` closes a confirmation like `Esc` (deliberate), so the "unrelated key" case now uses `G`. `UniqueName` naming extracted to `numberedName` (shared with trash naming); copy tests stayed green. `D` defaults to Cancel so Enter alone never deletes; `y` is required.
- **L12** Independent verifier launched in background for T5/T7/T8 (S5, S6, S8, S9, S12, S13), baseline 1b944a5. T9 RED: build failed (undefined `ValidateName`, `Rename`, `Mkdir`, `InputPending`). GREEN: `go test -race ./...` ok. Rename never replaces an existing entry; errors stay inside the prompt.
- **L13** T10 RED: build failed (undefined `HelpOpen`). GREEN: `go test -race ./...` ok. Runtime (tmux 120x30): help bar fits one line with "? help · q quit" right-aligned; `?` overlay shows Navigation/Selection/Files/Other in two columns. README added (install, keys, behavior table, WSL trash note).
