# Lógica Común — Materialización y Ejecución

> En workspace mode, `<roadmap-root>` se reemplaza por `<abs-roadmap-root>` y los comandos git usan `git -C <repo-path>`.

## Modelo

Roadmaps nuevos usan máximo dos niveles:

```text
<roadmap-root>/
├── O01-outcome/
│   ├── README.md
│   └── T001-task.md
└── T001-task-directa.md
```

## Auto-numbering

La raíz permite dos secuencias (`OXX` para Outcomes y `TXXX` para tasks directas), por eso no usar un único `schema.id.next` en la raíz cuando puede haber mezcla.

```bash
# Próximo Outcome en la raíz: listar directorios OXX-*
find <roadmap-root>/ -maxdepth 1 -type d -name 'O[0-9][0-9]-*' -printf '%f\n' | sort

# Próxima task directa en la raíz: listar archivos TXXX-*.md
find <roadmap-root>/ -maxdepth 1 -type f -name 'T[0-9][0-9][0-9]-*.md' -printf '%f\n' | sort

# Próxima task dentro de un Outcome: una sola secuencia activa, usar rootline
rootline describe <roadmap-root>/OXX-name/ --field schema.id.next
```

Tomar el mayor prefijo y sumar 1; si no hay ninguno, usar `O01` o `T001`.

## Verificación de padre

Antes de crear un archivo, verificar que el directorio destino existe:

```bash
rootline describe <directorio>/
```

Si no existe, informar al usuario y no crear archivos fuera del roadmap.

## Cascading links

Después de crear una task dentro de un Outcome, actualizar la tabla `## Tasks` del README del Outcome:

```markdown
| [TXXX](TXXX-task-name.md) | Descripción breve |
```

No agregar columna Estado; el estado se lee desde frontmatter.

## Dependencias

- Nueva convención: `[[blocked_by:T001-prerequisite]]` en la task bloqueada.
- Compatibilidad lectura-only: roadmaps viejos pueden contener `[[blocks:...]]`.

## Comandos Rootline de Referencia

| Comando | Cuándo usarlo |
|---------|---------------|
| `rootline validate <path>` | Después de crear/editar `.md` |
| `rootline fix <path>` | Si validate falla y la propuesta es segura |
| `rootline describe <dir> --field schema.id.next` | Auto-numbering |
| `rootline new <path>` | Crear archivo con frontmatter correcto |
| `rootline query <path> --where "expr"` | Listar tasks por metadata |
| `rootline tree <path> --where "expr" --output json` | Vista jerárquica con conteos |
| `rootline graph <path> --where "expr" --check` | Validar dependencias |

No usar `rootline stats`; `tree` ya incluye conteos.
