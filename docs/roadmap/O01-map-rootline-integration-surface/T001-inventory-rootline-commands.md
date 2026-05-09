---
estado: Completed
tipo: task
---
# T001: Inventory every Rootline CLI command and its flags that matter to Pi.

**Outcome**: [O01 Map Rootline integration surface](README.md)
**Contribuye a**: CE1 del Outcome.

## Preserva

- INV1: No Pi tool is implemented before its Rootline command contract and risk class are documented.
  - Verificar: Check downstream tasks reference this Outcome as source of truth.

## Contexto

Esta task forma parte de O01 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Inventory every Rootline CLI command and its flags that matter to Pi.

## Alcance

**In**:
1. Run ./rootline --help or rootline --help and capture the command list.
2. Record each command in a command inventory document inside this task.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.

## Criterios de Aceptación

- Run ./rootline --help or rootline --help and capture the command list.
- Record each command in a command inventory document inside this task.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `Rootline CLI commands in cmd/rootline/`
- `README.md`
- `docs/*.md`

## Inventario de comandos

### Comandos globales

| Comando | Descripción | Flags clave | Clase de riesgo |
|---------|-------------|-----------|-----------------|
| `rootline --version` | Muestra la versión del CLI | N/A | read-only |
| `rootline --help` | Muestra ayuda general | N/A | read-only |
| `rootline --output` | Formato de salida global | `json\|table` (default: json) | read-only |
| `rootline --field` | Extracción de dot-path | repetible | read-only |

### Comandos de análisis y diagnóstico

| Comando | Descripción | Flags clave | Clase de riesgo |
|---------|-------------|-----------|-----------------|
| `analyze` | Ejecuta detectores de inferencia en documentos | `--incremental`, `--threshold` | read-only |
| `describe` | Muestra el esquema .stem efectivo para un directorio | `--by-domain` | read-only |
| `explain` | Traza por qué un documento tiene un estado dado | N/A | read-only |
| `graph` | Construye y visualiza grafo de dependencias desde wiki-links | `--check`, `--format` (dot\|mermaid), `--open`, `--where` | read-only |
| `query` | Busca y filtra registros por expresiones | `--where`, `--count`, `--limit`, `--sort`, `--from` | read-only |
| `stats` | Resumen de conteos por tipo y estado | `--from`, `--where` | read-only |
| `trace` | Traversal BFS desde un archivo siguiendo wiki-links | `--depth`, `--format` (tree\|json), `--reverse`, `--type` | read-only |
| `tree` | Vista jerárquica con conteos de completitud | `--where` | read-only |
| `validate` | Valida documentos contra reglas .stem | `--all`, `--staged`, `--strict`, `--where` | read-only |

### Comandos de modificación de documentos

| Comando | Descripción | Flags clave | Clase de riesgo |
|---------|-------------|-----------|-----------------|
| `apply` | Aplica resultados de analyze a .stem y documentos | `--dry-run` | mutates-stem / mutates-docs |
| `fix` | Auto-repara errores de validación | `--all`, `--dry-run`, `--no-propagate` | mutates-docs |
| `set` | Establece campos de frontmatter o secciones | `--create`, `--dry-run`, `--no-validate` | mutates-docs |

### Comandos de scaffolding y migración

| Comando | Descripción | Flags clave | Clase de riesgo |
|---------|-------------|-----------|-----------------|
| `init` | Infiere schema .stem desde patrones de frontmatter | `--dry-run`, `--force`, `--template` | mutates-stem |
| `migrate` | Detecta cambios en .stem o realiza operaciones bulk | `--dry-run`, `--from`, `--rename`, `--scaffold`, `--split` | mutates-stem / mutates-docs |
| `new` | Crea documento nuevo con frontmatter pre-poblado | `--dry-run`, `--force` | mutates-docs |

### Comandos de integración

| Comando | Descripción | Flags clave | Clase de riesgo |
|---------|-------------|-----------|-----------------|
| `hooks` | Gestiona git pre-commit hook | subcomandos: install, status, uninstall | external |
| `hooks install` | Instala pre-commit hook | `--force` | external |
| `hooks status` | Verifica si hook está instalado | N/A | external |
| `hooks uninstall` | Desinstala pre-commit hook | N/A | external |
| `serve` | Inicia servidor MCP | `--addr`, `--stdio` | external |
| `completion` | Genera scripts de completion shell | shells: bash, zsh, fish | read-only |

### Notas sobre riesgo

- **read-only**: No modifica filesystem ni .stem
- **mutates-docs**: Modifica contenido de frontmatter o secciones en documentos
- **mutates-stem**: Modifica archivos .stem
- **external**: Interactúa con sistemas externos (git, MCP server, shell)
