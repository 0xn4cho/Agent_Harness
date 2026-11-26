# SquibCode 🐙

Agente multiagente de código, notas y DevOps para terminal, escrito en Go y pensado para Windows. SquibCode se conecta a Groq para interpretar tus peticiones y, con tu autorización, puede inspeccionar código, leer o editar archivos, buscar y mover ficheros, ejecutar comandos de PowerShell y consultar el estado de Docker o Kubernetes.

> SquibCode tiene acceso a herramientas locales. Revisa siempre la operación y los argumentos antes de aprobar una escritura, movimiento o comando.

## Qué puede hacer

- Orquestar turnos entre **tres roles especializados** (`leader`, `implementer`, `reviewer`) definidos en `agents/*.md`, cada uno con su propio modelo, límite de salida, esfuerzo de razonamiento y herramientas permitidas.
- Autocargar instrucciones del workspace (`SQUIB.md`, `.squib/instructions.md` o `06_Sistema/Reglas-Squib.md`) para respetar tus carpetas y rutas sin tener que repetirlas.
- Responder **directamente** (`DIRECT_ANSWER`) en consultas de solo lectura (por ejemplo, listar tareas pendientes o leer notas) sin disparar los roles de implementación ni revisión.
- Leer y escribir archivos dentro del workspace, y listar, buscar, copiar o mover rutas externas de forma segura con confirmación reforzada.
- Ejecutar comandos de PowerShell clasificados por nivel de riesgo (`low`, `high`, `blocked`) y consultar contenedores Docker o pods de Kubernetes.
- Trabajar con Groq o en modo local simulado (`--mock`), útil para probar la interfaz y el flujo sin consumir API.

## Interfaz Tokyo Night

- **Banner adaptativo `SQUIBCODE`:** fuente *ANSI Shadow* con degradado TrueColor azul → cian, sombra pizarra y destello metálico diagonal al abrir en terminales interactivas anchas, con fallback compacto en terminales estrechas o salida redirigida.
- **Cabecera de estado compacta:** muestra en dos líneas el workspace actual, si se cargaron reglas `.md` (`✓`), los modelos activos y los comandos rápidos.
- **Prompt y mensajería limpia:** prompt `❯` en violeta, cabeceras de rol diferenciadas por color (`● leader`, `● implementer`, `● reviewer`), spinner Braille contextual mientras trabaja y árboles discretos de herramientas (`├─` / `└─` con `✓` / `✗` sin JSON crudo).
- **Puerta de permisos en barra lateral:** formato compacto `│ Permiso · <tool> → <destino>` diferenciado por nivel de riesgo.
- **Terminal Windows nativa:** activa automáticamente UTF-8 (`65001`) y procesamiento VT al arrancar, y sale con código `0` en `exit`, `quit` y `Ctrl+C`.

## Requisitos

- Windows con PowerShell o Windows Terminal.
- [Go 1.27.3](https://go.dev/dl/) o una versión compatible.
- Una cuenta y API key de [Groq](https://console.groq.com/keys) para el modo real.
- Opcional: Docker CLI y/o `kubectl` para las consultas DevOps.

## Inicio rápido

### 1. Clona el repositorio

```powershell
git clone https://github.com/0xn4cho/Agent_Harness
cd Agent_Harness
```

### 2. Configura la clave de Groq

Para usarla solo en la sesión actual de PowerShell:

```powershell
$env:GROQ_API_KEY = "gsk_tu_clave_aqui"
```

Para guardarla de forma permanente en tu usuario de Windows (necesario si lanzas `squib.exe` desde un acceso directo o perfil de Windows Terminal):

```powershell
[System.Environment]::SetEnvironmentVariable("GROQ_API_KEY", "gsk_tu_clave_aqui", "User")
```

Cierra y abre una nueva terminal después de configurarla. No subas nunca esa clave a Git ni la guardes en archivos `.env` compartidos.

### 3. Ejecuta SquibCode

```powershell
go run .
```

Cada rol definido en `agents/` (`leader.md`, `implementer.md`, `reviewer.md`) configura su propio modelo y presupuesto:

| Rol | Modelo por defecto | `max_completion_tokens` | Herramientas |
|---|---|---|---|
| `leader` | `openai/gpt-oss-20b` | `600` | `read_file`, `list_directory`, `find_files` |
| `implementer` | `openai/gpt-oss-120b` | `1024` | `bash`, `copy_path`, `devops_status`, `find_files`, `list_directory`, `move_path`, `read_file`, `write_file` |
| `reviewer` | `openai/gpt-oss-20b` | `600` | `bash`, `devops_status`, `find_files`, `list_directory`, `read_file` |

Si necesitas forzar un único modelo global o ajustar el tope de salida de todos los roles:

```powershell
$env:GROQ_MODEL = "<nombre-del-modelo>"
$env:SQUIB_MAX_COMPLETION_TOKENS = "2048"
go run .
```

Para respetar los límites de tokens por minuto de Groq, SquibCode conserva como máximo 4500 tokens de historial, limpia bloques `<think>` antes de traspasar informes, reintenta automáticamente respuestas `429` respetando `Retry-After` y recupera en modo texto si un modelo restringido intenta llamar a una herramienta fuera de su rol (`tool_use_failed`).

### Agentes, reglas del workspace, informes y comandos

1. Si el workspace actual contiene `SQUIB.md`, `.squib/instructions.md` o `06_Sistema/Reglas-Squib.md`, SquibCode lo carga automáticamente en el prompt del sistema junto a la fecha local actual.
2. En consultas de solo lectura, `leader` lee los archivos reales y responde directamente (`DIRECT_ANSWER`) sin pasar por `implementer` ni `reviewer`.
3. En tareas de edición, creación o ejecución de comandos, el flujo avanza por `leader` → `implementer` → `reviewer`, guardando informes acotados en `progress/` (`leader.md`, `implementer.md`, `reviewer.md`). Si `reviewer` responde `CHANGES_REQUESTED`, el orquestador ejecuta automáticamente 1 ronda de corrección con `implementer` y vuelve a revisar.

Comandos disponibles en el prompt `❯`:

- `/help` · Muestra los comandos disponibles.
- `/roles` · Muestra los roles, modelos y herramientas activas.
- `/compact` · Resume y compacta el historial de sesión.
- `/clear` · Limpia el historial de sesión.
- `exit` / `quit` (o `Ctrl+C`) · Cierra SquibCode limpiamente con código `0`.

## Ejecutable y perfil en Windows Terminal

El repositorio ignora los binarios compilados (`*.exe` en `.gitignore`). Compila tu propio ejecutable en la raíz del proyecto:

```powershell
go build -o squib.exe .
.\squib.exe
```

Si creas un perfil exclusivo en **Windows Terminal** (con tema Tokyo Night `#1A1B26`, fuente `Cascadia Mono` y fondo personalizado opcional), apunta `commandline` directamente a `squib.exe` (sin `powershell.exe -NoExit`) y `startingDirectory` a la carpeta de tu workspace o vault para que la ventana se cierre automáticamente al escribir `quit`, `exit` o pulsar `Ctrl+C`:

```json
{
  "name": "SquibCode",
  "commandline": "C:\\ruta\\a\\Squib\\squib.exe",
  "startingDirectory": "C:\\ruta\\a\\tu\\workspace",
  "background": "#1A1B26",
  "foreground": "#C0CAF5",
  "cursorColor": "#BB9AF7"
}
```

## Modo local de prueba

El modo mock permite comprobar la interfaz y el flujo sin API key ni llamadas de red:

```powershell
go run . --mock
```

También puedes activarlo con una variable de entorno:

```powershell
$env:SQUIB_PROVIDER = "mock"
go run .
```

## Herramientas disponibles

| Herramienta | Uso |
|---|---|
| `read_file` | Lee un archivo dentro del workspace. |
| `write_file` | Crea o reemplaza un archivo dentro del workspace. |
| `bash` | Ejecuta un comando de PowerShell desde el workspace. |
| `devops_status` | Muestra el estado de Docker, Kubernetes o ambos. |
| `list_directory` | Lista el contenido de un directorio seleccionado (incluso fuera del workspace). |
| `find_files` | Busca archivos recursivamente por patrón (por ejemplo `*.md` o `*.pdf`). |
| `copy_path` | Copia un archivo o directorio sin sobrescribir destinos existentes. |
| `move_path` | Mueve o reubica un archivo o directorio sin sobrescribir destinos existentes. |

> **Alcance DevOps:** `devops_status` es de solo lectura: ejecuta únicamente `docker ps` y `kubectl get pods -A`. No valida ni despliega nada por sí misma. Valida siempre manifiestos generados antes de aplicarlos (`docker compose config`, `kubectl apply --dry-run=client -f <fichero>`).

## Seguridad y permisos

- Todas las herramientas solicitan confirmación explícita; los comandos de shell no permitidos en lectura y las operaciones externas requieren escribir `yes` o `si` completo y nunca permiten aprobación permanente (`always`).
- Las rutas de lectura/escritura en el workspace bloquean escapes por traversal, symlinks/junctions y ficheros sensibles (`.ssh`, `.aws`, `.kube`, `.env*`, etc.); las rutas externas también bloquean ubicaciones de credenciales y sobrescrituras.
- Las claves y tokens conocidos se eliminan del entorno de procesos secundarios (`bash` / `devops_status`).
- Las salidas de comandos, archivos y respuestas HTTP tienen límites de tamaño y timeout por turno de 5 minutos.

## Kubernetes opcional

Por defecto, SquibCode no usa automáticamente el kubeconfig global. Si necesitas consultar un clúster, indica un kubeconfig explícito:

```powershell
$env:SQUIB_KUBECONFIG = "C:\ruta\al\kubeconfig"
go run .
```

## Desarrollo

Ejecuta la suite de pruebas y el análisis estático:

```powershell
go test ./...
go vet ./...
```

Estructura principal del proyecto:

```text
.
├── main.go                 # REPL, comandos slash, reglas de workspace y orquestador
├── agents/                 # Definiciones Markdown de leader, implementer y reviewer
├── progress/               # Informes de traspaso en tiempo de ejecución (ignorado en Git)
└── internal/
    ├── agents/             # Carga de roles (.md) e informes en progress/
    ├── api/                # Tipos compartidos de mensajes y herramientas
    ├── compact/            # Ventana de contexto y estrategias de compactación
    ├── permission/         # Puerta de permisos en barra lateral y políticas por riesgo
    ├── provider/           # Proveedor Groq (con retry 429 y tool_use_failed) y mock
    ├── security/           # Validación de rutas, clasificación de comandos y saneamiento
    ├── tool/               # Herramientas de archivos, shell, DevOps y rutas externas
    └── ui/                 # Tema Tokyo Night, banner animado, spinner y estilos ANSI
```
