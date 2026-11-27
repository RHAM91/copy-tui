# copiar

Un gestor de archivos de terminal con doble panel y teclas Vim: navega dos carpetas lado a lado y presiona <kbd>space</kbd> para copiar de una a la otra, en cualquier dirección.

## Inicio rápido

1. Instalación: usa el instalador del catálogo privado de actualizaciones (te pide el nombre de la app, `copiar-tui`, y tu token de descarga `cat_dl_...`), o compila con Go 1.24+:

   ```bash
   # Linux y macOS
   curl -fsSL https://updates.r0ma.cloud/install.sh | sh

   # Windows (PowerShell)
   irm https://updates.r0ma.cloud/install.ps1 | iex

   # desde el código fuente
   go install github.com/RHAM91/copy-tui/cmd/copiar@latest
   ```

   El instalador deja `copiar` en `~/.local/bin` (asegúrate de que esté en tu `PATH`) y guarda tu token en el perfil de tu shell como `COPIAR_TUI_UPDATE_TOKEN`.

   > **macOS:** los binarios no están firmados, por lo que la primera ejecución de un archivo descargado con el navegador es bloqueada por Gatekeeper. Permítelo una vez con `xattr -d com.apple.quarantine ./copiar`. Las actualizaciones con `copiar update` no se ven afectadas.

2. Ejecútalo en cualquier carpeta; ambos paneles se abren ahí:

   ```bash
   copiar            # con iconos Nerd Font
   copiar --no-icons # si la fuente de tu terminal no tiene glifos Nerd Font
   copiar --version  # muestra la versión instalada
   copiar update     # instala la última versión
   ```

3. Ve al destino en un panel, cambia al otro con <kbd>tab</kbd>, selecciona un archivo y presiona <kbd>space</kbd>.

Presiona <kbd>?</kbd> dentro de la app para ver todas las teclas.

## Teclas

| Navegación | | Archivos | |
|---|---|---|---|
| `j` / `k` | abajo / arriba | `space` | copiar al otro panel |
| `h` / `l` | carpeta superior / abrir carpeta | `x` | mover al otro panel (pregunta antes) |
| `gg` / `G` | primera / última entrada | `d` | mover a la papelera (pregunta antes) |
| `ctrl+d` / `ctrl+u` | media página abajo / arriba | `D` | eliminar permanentemente (pregunta antes, Cancelar por defecto) |
| `tab` | cambiar de panel | `r` | renombrar |
| `/`, `n` / `N` | buscar, coincidencia siguiente / anterior | `a` | nueva carpeta |
| `.` | mostrar / ocultar archivos ocultos | `?` | mostrar / ocultar ayuda |
| `ctrl+s` | guardar las rutas de ambos paneles como marcador | `q` | salir |
| `ctrl+b` | abrir la tabla de marcadores | | |
| `=` | misma carpeta en ambos paneles | | |
| `ctrl+t` | elegir un tema | | |

**Selección:** `m` marca la entrada y baja, `v` selecciona un rango (presiona `v` otra vez para conservarlo), `esc` limpia la selección. Copiar, mover y eliminar se aplican a la selección, o a la entrada bajo el cursor cuando no hay nada seleccionado.

## Cómo se comportan las operaciones

| Tema | Comportamiento |
|---|---|
| Semántica de copia | Como `cp -a`: las carpetas se copian de forma recursiva, los enlaces simbólicos se copian como enlaces (nunca se siguen) y se conservan los permisos y las fechas de modificación. |
| Conflictos de nombre | Un diálogo ofrece **Sobrescribir**, **Omitir** o **Renombrar** (`nombre (1).ext`), con la opción *Aplicar a todos*. Sobrescribir una carpeta combina su contenido. |
| Progreso y cancelación | Las operaciones largas muestran un modal de progreso; `esc` las cancela. Los archivos se escriben primero con un nombre temporal, así que una copia cancelada nunca deja archivos a medias ni destruye el archivo que iba a reemplazar. |
| Diseño | Los paneles van lado a lado; cuando la terminal tiene menos de 100 columnas se apilan: el panel izquierdo arriba y el derecho abajo. |
| Temas | `ctrl+t` lista **Catppuccin Mocha** (por defecto), **Kanagawa Wave**, **Kanagawa Dragon** y **Tokyo Night**. Al moverte por la lista se previsualiza cada tema; `enter` lo conserva y lo guarda, `esc` vuelve al anterior. Los temas colorean texto, bordes y resaltados y respetan el fondo de tu terminal. La elección se guarda como `theme = "kanagawa-wave"` en `config.toml`, junto a los marcadores (`~/.config/copiar/config.toml` en Linux). Un tema desconocido o un archivo ilegible vuelve al tema por defecto con un mensaje, y un archivo corrupto nunca se sobrescribe. |
| Misma carpeta | Copiar a la carpeta donde ya vive el elemento crea un duplicado `nombre (1).ext`. |
| Mover | Renombra dentro de un mismo sistema de archivos; entre sistemas de archivos copia y luego elimina el origen, así que un movimiento cancelado conserva el origen. |
| Actualizaciones | Define `COPIAR_TUI_UPDATE_TOKEN` con tu token de descarga para habilitarlas. Cuando existe una versión más reciente, la barra de estado muestra `vX.Y.Z available — run copiar update` (se comprueba en segundo plano al iniciar; no se muestra nada sin conexión ni sin token). `copiar update` descarga la versión para tu plataforma desde el catálogo de actualizaciones, verifica su tamaño, SHA-256 y firma Ed25519 con la clave pública incluida en el binario, y reemplaza el binario; usa `sudo copiar update` (o una terminal de Administrador en Windows) si está en una carpeta del sistema. Las compilaciones desde el código fuente (`dev`) no se actualizan. |
| Misma carpeta en ambos paneles | `=` pregunta en qué dirección ir: **Izquierda → Derecha** abre la carpeta izquierda en el panel derecho, **Derecha → Izquierda** lo contrario. La carpeta del panel activo viene preseleccionada, el foco se queda donde estaba y el panel que cambia coloca su cursor en la misma entrada. Cuando ambos paneles ya muestran la misma carpeta, solo se muestra un mensaje. |
| Marcadores | `ctrl+s` pide un nombre y guarda las carpetas de los paneles izquierdo y derecho; si el nombre ya existe, pregunta antes de reemplazarlo. `ctrl+b` abre la tabla: `enter` abre ambas carpetas, `r` renombra y `d` elimina (ambas preguntan antes). Antes de abrir se comprueban ambas carpetas: si una ya no existe puedes **Ir de todos modos** (solo se abre la que existe), **Eliminar** el marcador roto o cancelar; si ninguna existe, puedes eliminarlo o cancelar. Los marcadores se guardan en `~/.config/copiar/bookmarks.json` en Linux, `~/Library/Application Support/copiar/` en macOS y `%AppData%\copiar\` en Windows. |
| Papelera | `d` usa la papelera del sistema para poder restaurar los elementos: la papelera de freedesktop.org en Linux (`~/.local/share/Trash`), Finder en macOS (con **Put Back**; recurre a `~/.Trash` cuando no se puede usar Finder, por ejemplo por SSH) y la Papelera de reciclaje en Windows. |

> **Nota sobre WSL:** la papelera es la de Linux. Los elementos enviados a la papelera desde `/mnt/c/...` no aparecen en la Papelera de reciclaje de Windows.

> **Nota sobre tmux:** `ctrl+b` es el prefijo por defecto de tmux, así que tmux se lo queda. Presiónalo dos veces (`ctrl+b ctrl+b`) para enviarlo a copiar, o cambia el prefijo (por ejemplo `set -g prefix C-a`).

## Desarrollo

```bash
go test -race ./...          # pruebas unitarias y de interfaz
go run ./cmd/copiar          # ejecutar desde el código fuente
```

| Ruta | Responsabilidad |
|---|---|
| `cmd/copiar` | flags e inicio del programa |
| `internal/fsops` | operaciones de archivos (listar, copiar, mover, papelera, eliminar, renombrar), sin interfaz; el código específico de cada SO está en `*_linux.go`, `*_darwin.go`, `*_windows.go`, `*_unix.go` |
| `internal/bookmarks` | pares de carpetas izquierda/derecha con nombre, guardados como JSON en el directorio de configuración del usuario |
| `internal/config` | preferencias (el tema) en `config.toml`, en el directorio de configuración del usuario |
| `internal/updater` | cliente del catálogo de actualizaciones: comprobación de versión, descarga y verificación SHA-256 y Ed25519 |
| `internal/update` | servicio de autoactualización sobre el cliente del catálogo |
| `internal/ui` | modelo de Bubble Tea: paneles, diálogos, teclas, tema (Catppuccin Mocha) |

## Publicación de versiones

Las versiones se publican en el catálogo privado de actualizaciones con `.github/workflows/release.yml` cuando se sube un tag de versión:

```bash
git tag v0.6.0 && git push origin v0.6.0
```

El workflow ejecuta `go test -race`, compila los cuatro targets habilitados (`linux/amd64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`), firma cada binario y los publica juntos; el catálogo solo expone una versión cuando todos los targets están subidos. El tag debe tener la forma `vMAJOR.MINOR.PATCH`, y el catálogo rechaza una versión ya publicada o menor. La lista `TARGETS` del workflow debe coincidir con los targets marcados en el panel del catálogo.

Secrets requeridos del repositorio: `CATALOG_URL`, `CATALOG_UPLOAD_TOKEN`, `CATALOG_SIGNING_KEY`.
