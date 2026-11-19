# layout-and-themes — stacked panels on narrow terminals and a ctrl+t theme picker

Objective: stack the panels vertically on small terminals, and add selectable themes (Kanagawa Dragon, Kanagawa Wave, Tokyo Night) chosen with `ctrl+t` and persisted in `config.toml`.
Branch: `feat/responsive-themes` (base `main` @ 41b66ec) · Delivery: `ask-on-risk` · Test runner: `go test ./...` (stdlib `testing`, model driven through `Update`).

## Specs

- **S1** Responsive: "Los paneles deben ser responsivos es decir si se detecta un tamaño de panalla pequeño los paneles deben mostrarse uno arriba y el otro abajo, y en pantalla normal o grande como lo hace hoy."
- **S2** Breakpoint: "vamos con la a" — small means fewer than 100 columns.
- **S3** Themes: "Añade selección de temas, para que persista añade un archivo config.toml en el que se pueda guardar dicha preferencia, los temas que necesito que añadas además del que ya está es el: Kanagawa dragon, Kanagawa wave, y Tokio Nigth".
- **S4** Picker: "debe haber un shortcut ctr+t y mostrar una ventana para poder elegir los temas."
- **S5** Live preview: "la a" — moving through the list repaints the app with that theme; `enter` saves it, `esc` restores the previous theme.
- **S6** Background: "vamos con el a" — themes change text, border, cursor and dialog colors only; the terminal background stays.
- Assumption A1: stacked panels split the body height in half; the left panel is on top.
- Assumption A2: `config.toml` lives next to `bookmarks.json` (`<os.UserConfigDir()>/copiar/config.toml`) with `theme = "<id>"`; ids are `catppuccin-mocha` (default, the current theme), `kanagawa-wave`, `kanagawa-dragon`, `tokyo-night`. It is written atomically; an unreadable or corrupt file is reported and never overwritten; an unknown theme id falls back to the default with a status message.
- Assumption A3: TOML is parsed with `github.com/BurntSushi/toml` (new dependency).

## Tasks

| ID | Specs | Task | Route | Risk | Status | Commit |
|----|-------|------|-------|------|--------|--------|
| T1 | S1,S2,A1 | Stack panels under 100 columns; per-panel heights | inline | none | done | 7b9c89b |
| T2 | S3,S6 | Palettes as data, styles rebuilt by `applyTheme` | inline | none | done | ef62050 |
| T3 | S3,A2,A3 | `internal/config`: load/save `config.toml` | inline | item 5 (new dependency) | done | 530881b |
| T4 | S4,S5 | `ctrl+t` picker with live preview, saves on enter; main wiring | inline | none | done | af0d0e1 |
| T5 | all | Help, README, manual check in tmux | inline | none | done | af0d0e1 |

Route evidence: no Writer trigger (tasks build on each other, one session), so the parent writes inline.

## Log

- **L1** (2026-11-19) User request: "excelente hermano, veo que funciona perfecto. Añade 2 features más. Feature 1: Los paneles deben ser responsivos es decir si se detecta un tamaño de panalla pequeño los paneles deben mostrarse uno arriba y el otro abajo, y en pantalla normal o grande como lo hace hoy. Feature 2: Añade selección de temas, para que persista añade un archivo config.toml en el que se pueda guardar dicha preferencia, los temas que necesito que añadas además del que ya está es el: Kanagawa dragon, Kanagawa wave, y Tokio Nigth, debe haber un shortcut ctr+t y mostrar una ventana para poder elegir los temas. De igual manera hazme las preguntas necesarias para mejorar la implementación"
- **L2** Breakpoint answer: "vamos con la a" (fewer than 100 columns).
- **L3** Preview answer: "la a" (live preview, enter saves, esc restores).
- **L4** Background answer: "vamos con el a" (colors only, terminal background kept).
- **L5** T1: RED (narrowing kept one row of panels) → GREEN, 3 layout tests; tmux at 80×24 shows the panels stacked.
- **L6** T2: RED (undefined `Themes`) → GREEN, 3 theme tests. Differential check: the default theme renders byte-identical (TrueColor, panels + delete dialog + help) to the code before the refactor. The progress bar gradient is rebuilt per theme.
- **L7** T3: RED (undefined `DefaultPath`) → GREEN, 6 config tests. `SetTheme` keeps unknown keys; a corrupt file is never overwritten. New dependency `github.com/BurntSushi/toml v1.4.0`: `go mod verify` passes and it pulls no other module. The dependency check ran inline instead of through a separate verifier.
- **L8** T4/T5: RED (undefined `ThemePickerOpen`) → GREEN, 6 picker tests plus a modal-precedence case; `go test -race ./...` green; Windows and macOS builds compile. Manual tmux check with an isolated `XDG_CONFIG_HOME`: ctrl+t lists the 4 themes with swatches, j previews Kanagawa Wave (border #957FB8), enter writes `theme = "kanagawa-wave"`, and a restart starts in Kanagawa Wave.
- **L9** Branch total ≈ 955 authored changed lines over the 400 heuristic, in 4 work-unit commits; delivery strategy to be confirmed with the user before the PR.
