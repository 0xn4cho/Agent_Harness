# SquibCode 🫍

Agente de código y DevOps para terminal, escrito en Go y pensado para Windows. Squib se conecta a Groq para interpretar tus peticiones y, con tu autorización, puede inspeccionar código, crear o modificar archivos dentro del proyecto, ejecutar comandos de PowerShell y consultar el estado de Docker o Kubernetes.

> Squib es un agente con acceso a herramientas locales. Revisa siempre la operación y el diff antes de aprobar una escritura o un comando.

## Qué puede hacer

- Leer archivos del workspace para entender el proyecto.
- Crear y actualizar archivos de código, configuración o documentación.
- Ejecutar comandos de PowerShell para compilar, formatear, ejecutar tests o consultar Git.
- Consultar contenedores Docker y pods de Kubernetes.
- Trabajar con Groq o en modo local simulado (`--mock`), útil para probar el flujo sin consumir API.

## Requisitos

- Windows con PowerShell.
- [Go 1.27.3](https://go.dev/dl/) o una versión compatible.
- Una cuenta y una API key de [Groq](https://console.groq.com/keys) para usar el modelo real.
- Opcional: Docker CLI y/o `kubectl` si vas a usar las consultas DevOps.

## Inicio rápido

### 1. Clona el repositorio

```powershell
git clone <URL-DEL-REPOSITORIO>
cd Squib
```

### 2. Configura la clave de Groq

Para usarla solo en la ventana actual de PowerShell:

```powershell
$env:GROQ_API_KEY = "gsk_tu_clave_aqui"
```

Para guardarla para tu usuario de Windows:

```powershell
[System.Environment]::SetEnvironmentVariable("GROQ_API_KEY", "gsk_tu_clave_aqui", "User")
```

Cierra y abre una nueva terminal después de usar la configuración permanente. No subas nunca esa clave a Git ni la guardes en archivos `.env` que vayan a compartirse.

### 3. Ejecuta Squib

```powershell
go run .
```

El modelo predeterminado es `openai/gpt-oss-120b`. Para seleccionar otro modelo disponible en Groq:

```powershell
$env:GROQ_MODEL = "<nombre-del-modelo>"
go run .
```

### 4. Habla con el agente

Al aparecer el prompt, escribe una petición en lenguaje natural. Por ejemplo:

```text
Revisa main.go y explícame cómo funciona el bucle del agente.
```

```text
Crea un archivo hello.py con una función que salude y sus tests con pytest.
```

```text
Ejecuta los tests de Go y explícame cualquier fallo.
```

Para salir, escribe `exit` o `quit`.

## Ejecutable opcional

Puedes compilar un binario para no depender de `go run`:

```powershell
go build -o squib.exe .
.\squib.exe
```

## Modo local de prueba

El modo mock permite comprobar la interfaz y el flujo del agente sin API key ni llamadas de red:

```powershell
go run . --mock
```

También se puede activar con una variable de entorno:

```powershell
$env:SQUIB_PROVIDER = "mock"
go run .
```

## Cómo funciona

1. Escribes una petición en la terminal.
2. Squib la envía, junto con el contexto de conversación y las herramientas disponibles, al modelo de Groq.
3. El modelo puede responder directamente o solicitar una herramienta, por ejemplo `read_file`, `write_file`, `bash` o `devops_status`.
4. Squib muestra la herramienta y sus argumentos; debes aprobar o denegar la acción antes de ejecutarla.
5. El resultado vuelve al modelo, que puede continuar trabajando o darte la respuesta final.

El agente tiene un máximo de rondas por turno, un timeout global de cinco minutos y una ventana de contexto acotada por mensajes y tokens para evitar bucles, consumo de memoria y costes descontrolados.

## Herramientas disponibles

| Herramienta | Uso |
|---|---|
| `read_file` | Lee un archivo del workspace. |
| `write_file` | Crea o reemplaza un archivo del workspace. |
| `bash` | Ejecuta un comando de PowerShell desde el workspace. |
| `devops_status` | Muestra el estado de Docker, Kubernetes o ambos. |

Ejemplos de peticiones:

```text
Lee internal/tool/bash.go y busca posibles problemas de seguridad.
```

```text
Crea una función en Go para validar un correo y añade tests de tabla.
```

```text
Consulta el estado de Docker y resume los contenedores activos.
```

## Seguridad y permisos

Squib aplica varias protecciones antes de ejecutar herramientas:

- Todas las herramientas solicitan confirmación explícita.
- Los comandos de shell de riesgo requieren escribir `yes` completo; las acciones sensibles no se pueden aprobar permanentemente.
- La lectura y escritura de archivos se limita al workspace y bloquea rutas habituales de credenciales, como `.ssh`, `.aws`, `.kube` y `.env`.
- Las claves y tokens conocidos se eliminan del entorno de los procesos secundarios.
- Las salidas de comandos, archivos y respuestas HTTP tienen límites de tamaño; el historial se compacta para controlar el contexto.

Estas medidas no sustituyen el criterio humano: aprobar un comando permite que se ejecute con los permisos de tu usuario de Windows.

## Kubernetes opcional

Por defecto, Squib no usa automáticamente el kubeconfig global. Si necesitas consultar un clúster, indica de forma explícita un kubeconfig permitido:

```powershell
$env:SQUIB_KUBECONFIG = "C:\ruta\al\kubeconfig"
go run .
```

No compartas ni subas ese fichero: puede contener credenciales del clúster.

## Desarrollo

Ejecuta la suite de pruebas:

```powershell
go test ./...
```

Comprueba problemas estáticos:

```powershell
go vet ./...
```

La estructura principal del proyecto es:

```text
.
├── main.go                 # REPL y bucle principal del agente
└── internal/
    ├── api/                # Tipos compartidos de mensajes y herramientas
    ├── compact/            # Ventana de contexto y estrategias de compactación
    ├── permission/         # Confirmación y política de permisos
    ├── provider/           # Proveedor Groq y proveedor mock
    ├── security/           # Validación de rutas y saneamiento de entorno
    ├── tool/               # Herramientas de archivos, shell y DevOps
    └── ui/                 # Estilos y banner de terminal
```

