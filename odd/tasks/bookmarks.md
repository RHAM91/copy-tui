# bookmarks — save and reopen recurring panel paths

Objective: save the left/right panel paths under a name (`ctrl+s`) and reopen them from a bookmark table (`ctrl+b`), handling broken bookmarks.
Branch: `feat/bookmarks` (base `main` @ 41e0c28) · Delivery: `ask-on-risk` · Test runner: `go test ./...` (stdlib `testing`, model driven through `Update`).

## Specs

- **S1** Purpose: "Necesito que tenga la opción de bookmarks, es decir que pueda guadar rutas recurrentes".
- **S2** Save: "si estoy parado en la ruta (panel a y panel b) con los paths que necesito, que exita el comando ctr+s y que me pregunte el nombre del bookmark". A bookmark stores the left and right panel directories.
- **S3** Duplicate name: "la 1, que pregunte si quiero reemplazarlo" — saving with an existing name asks for confirmation before replacing it.
- **S4** Open: "en cualquiera que sea el lugar donde me encuentre al presionar ctrl+b me mostrará la tabla de bookmarks, seleccionaré la deseada y al presionar enter colocará en los paneles las rutas".
- **S5** Broken: "en dado caso una o ambas rutas ya no existan, antes de dirigirse debe evualuar su existencia si no existe entonces debe avisar con un mensaje claro y la opción de borrar ya que el bookmark está roto".
- **S6** Partially broken: "la 1, que me deje ir igual" — when only one path is missing, the dialog names it and offers go anyway (open only the existing path), delete, or cancel.
- **S7** Table actions: "2 con confirmarción de ambas" — in the table, `d` deletes and `r` renames; both ask for confirmation.
- Assumption A1: bookmarks live in `<os.UserConfigDir()>/copiar/bookmarks.json` (Linux `~/.config/copiar/bookmarks.json`), written atomically; a corrupt file is reported and never overwritten.
- Assumption A2: names are trimmed, must be non-empty, and are compared case-insensitively for duplicates; the table is sorted by name.
- Assumption A3: `ctrl+b` works from the normal panel mode (not while another dialog is open); the README notes it clashes with tmux's default prefix.
- Assumption A4: rename to a name used by another bookmark is refused with an error in the prompt.

## Tasks

| ID | Specs | Task | Route | Risk | Status | Commit |
|----|-------|------|-------|------|--------|--------|
| T1 | S1,S2,A1,A2 | `internal/bookmarks` store: load, save atomically, add/replace, rename, delete | inline | none | done | 917f42d |
| T2 | S2,S3 | `ctrl+s` name prompt, replace confirmation | inline | none | done | see L7 |
| T3 | S4,S5,S6 | `ctrl+b` table, enter navigates, broken dialog (go anyway / delete / cancel) | inline | none | done | see L7 |
| T4 | S7,A4 | `d` delete and `r` rename in the table, with confirmations | inline | none | done | see L7 |
| T5 | all | Help, README, manual check in tmux | inline | none | done | see L7 |

Route evidence: no Writer trigger reason (tasks depend on each other), so the parent writes inline. The bookmarks file is a new persisted record, not a change to existing data, so no high-risk item applies.

## Log

- **L1** (2026-11-18) User request: "quiro añadir una feature nueva, la feature es: Necesito que tenga la opción de bookmarks, es decir que pueda guadar rutas recurrentes, entonces si estoy parado en la ruta (panel a y panel b) con los paths que necesito, que exita el comando ctr+s y que me pregunte el nombre del bookmark, una vez guardado, en cualquiera que sea el lugar donde me encuentre al presionar ctrl+b me mostrará la tabla de bookmarks, seleccionaré la deseada y al presionar enter colocará en los paneles las rutas, en dado caso una o ambas rutas ya no existan, antes de dirigirse debe evualuar su existencia si no existe entonces debe avisar con un mensaje claro y la opción de borrar ya que el bookmark está roto. Me puedes ayudar a añadir esa caracteristica? usa odd y hazme las preguntas necesarias para mejorar la implementación, también te comento que estamos en tmux asi que puedes crear sesiones nuevas para hacer pruebas."
- **L2** Duplicate name answer: "la 1, que pregunte si quiero reemplazarlo".
- **L3** Partially broken answer: "la 1, que me deje ir igual".
- **L4** Table actions answer: "2 con confirmarción de ambas".
- **L5** The user's tmux prefix is `C-a`, so `ctrl+b` reaches the app in their setup.
- **L6** T1: RED (build failed, undefined `Bookmark`) → GREEN, 9 store tests. A corrupt file makes every write fail, so it is never overwritten.
- **L7** T2–T5 in one commit (one UI unit: the table opens the delete/rename dialogs, so they are not independent). RED (undefined `BookmarksOpen`) → GREEN, 13 UI tests plus 3 modal-precedence cases; `go test -race ./...` green; Windows and macOS cross-builds compile. The manual check in a separate tmux session with an isolated `XDG_CONFIG_HOME` covered save, open, partly broken + go anyway, fully broken (enter defaults to Cancel) + delete, replace, and rename. It found a bug: `padRight` always adds a space, so the highlighted row was one column too wide and wrapped once styled. Pinned by `TestBookmarkTableRowsWithLongPathsDoNotWrap` (it forces TrueColor, because the wrap only shows with ANSI styles), RED before the fix. The broken dialog shows full wrapped paths instead of truncated ones, so the message is clear. Authored lines ≈ 1280 across both commits, over the 400 heuristic; kept as two units because the UI is one behavior. Risk: none (new keys and a new file; no existing data, flags, or shared helpers changed).
