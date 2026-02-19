> **Superseded**: This is the original research document. The project was renamed
> from "docrules" to **Rootline**, and the rules file from `.docrules` to **`.stem`**.
> The schema model was also redesigned (see D19 in [intent doc](../intent/v0-rootline.md)).
> This document is preserved as historical record.

# docrules: File-Based Documentation Indexer

**Fecha**: 2026-02-16
**Actualizado**: 2026-02-16 (sesion de analisis competitivo y tech stack)
**Contexto**: Explorar la idea de un CLI standalone que centralice la logica determinista de documentacion (parsing, validacion, queries, derivacion de estado) usando archivos `.docrules` por directorio con herencia parent→child, al estilo `.htaccess`.
**Origen**: Sesion de retrospectiva donde se identifico tracking drift causado por logica dispersa en multiples skills/hooks.

---

## 1. Problema

### Logica Determinista Dispersa

En el homeserver automation, la logica para gestionar documentos markdown esta dispersa en multiples consumers:

| Consumer | Que hace | Como lo hace |
|----------|----------|--------------|
| `/service` skill | Buscar Task por ID, validar tipo | `find docs/epics -name "T*.md"` + `grep "servicio-docker"` |
| `/module` skill | Buscar Task, validar tipo | Mismo patron, distinto tipo |
| `/operation` skill | Buscar Task, validar tipo | Mismo patron, distinto tipo |
| `/instance` skill | Buscar Task, validar tipo | Mismo patron, distinto tipo |
| `/roadmap view` | Derivar estado de Stories/Features | Glob + read frontmatter + logica inline |
| `/pendings` | Listar PRDs por estado | `grep "Estado" docs/prd/` |
| Write hook | Validar frontmatter de Tasks | Prompt-based parsing inline |

**Cada consumer re-implementa**: discovery (donde esta el archivo), parsing (como extraer metadata), validacion (que valores son validos), y en algunos casos derivacion (calcular estado padre desde hijos).

### Incidente Concreto

Tracking drift en E02/F01: 3 Tasks estaban `Completado` en sus archivos pero el README del Feature mostraba 0%. Causa raiz: el estado vivia en dos lugares (Task file + README tabla) y ningun hook validaba consistencia entre ambos. Se resolvio migrando a YAML frontmatter como single source of truth, pero la logica de derivacion sigue inline en el skill `/roadmap view`.

### Consecuencias

- **Fragilidad**: Si el formato de metadata cambia, hay que actualizar N skills/hooks
- **Inconsistencia**: Cada consumer parsea ligeramente distinto
- **No escalable**: Agregar un nuevo tipo de documento = tocar multiples archivos
- **No reutilizable**: La logica es especifica de este proyecto, no se puede compartir

---

## 2. Objetivo Estrategico

### El Proyecto ES la Carta de Presentacion

docrules no es un generador de CV ni un portfolio builder. **El proyecto publicado como open source ES la carta de presentacion profesional.** Un CLI bien ejecutado y publicado demuestra:

- **Diseno de software**: Arquitectura limpia, per-directory inheritance, separacion de concerns
- **Dominio de tooling moderno**: MCP server nativo, CLI idiomatico
- **Resolucion de problema real**: Nace de necesidad concreta, no es un toy project
- **Capacidad de publicar**: Documentacion, CI/CD, distribucion, dogfooding

### Audiencia Target

El proyecto debe resonar simultaneamente con tres audiencias profesionales:

| Audiencia | Que valora | Como docrules lo demuestra |
|-----------|------------|---------------------------|
| **DevOps / SRE / Platform** | Go, tooling, pragmatismo | CLI en Go como kubectl/terraform |
| **Software Engineering** | Clean code, testing, design patterns | Arquitectura modular, tests, CI/CD |
| **AI/LLM Engineering** | Integracion AI nativa, MCP | MCP server en Go (diferenciador vs TS default) |

### Principio de Rentabilidad

La eleccion de tecnologia debe maximizar **ROI profesional = impacto / tiempo invertido**. No se busca la tecnologia "mas impresionante" sino la que produce el mejor resultado publicable en el menor tiempo.

---

## 3. Concepto

### Vision

Un **CLI standalone** (binario compilado) que:
1. Lee archivos `.docrules` por directorio (con herencia parent→child)
2. Indexa documentos markdown con su metadata (YAML frontmatter o inline)
3. Valida documentos contra las reglas de su directorio
4. Responde queries estructuradas (buscar, filtrar, derivar estado)
5. Produce output en JSON para consumo programatico
6. Expone MCP server para integracion nativa con AI assistants

### Modelo de Herencia: parent→child (estilo .htaccess)

docrules usa herencia **parent→child**: las reglas se definen en directorios padre y se propagan hacia abajo a los hijos. Cada nivel hijo puede hacer override o restringir reglas heredadas.

**Esto es diferente de `.editorconfig`** que funciona child→parent (un archivo busca hacia arriba preguntando "que reglas me aplican"). docrules funciona como `.htaccess`: un directorio dice "mis hijos cumplen estas reglas".

La razon: en una jerarquia Epic→Feature→Story→Task, las reglas se definen a nivel Epic/Feature y se heredan a Stories/Tasks. El padre impone estructura a sus descendientes.

```
docs/
├── .docrules              # Reglas base (heredan hacia abajo a todo)
├── epics/
│   ├── .docrules          # Reglas para epics (hereda de padre + override)
│   ├── E01-infra/
│   │   ├── F01-bootstrap/
│   │   │   ├── S001-setup/
│   │   │   │   ├── .docrules    # Reglas para Tasks (hereda + override)
│   │   │   │   └── T001-xxx.md
```

Campos no redefinidos en un hijo se heredan automaticamente del padre. Un `.docrules` con `root: true` marca el tope de la cadena de herencia.

### Separacion de Responsabilidades

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────┐
│ Skills      │────>│                  │────>│ docs/*.md   │
│ Hooks       │────>│  [app] CLI/MCP   │────>│ .docrules   │
│ /roadmap    │────>│  (single binary) │     └─────────────┘
│ /pendings   │────>│                  │
└─────────────┘     └──────────────────┘
  thin clients        logica central         data + config
```

Los skills dejan de tener logica de parsing/validacion y se convierten en orquestadores de intencion que preguntan a la app.

---

## 4. Estado del Arte

### A. Markdown-as-Database (indexacion estructural)

Herramientas que convierten archivos markdown en datos queryables.

| Herramienta | Lenguaje | Backend | Frontmatter | Query | Per-dir rules | Estado |
|-------------|----------|---------|-------------|-------|---------------|--------|
| [MarkdownDB (mddb)](https://github.com/datopian/markdowndb) | Node.js | SQLite | Si | SQL + JS API | No | Activo |
| [markdown-to-sqlite](https://github.com/simonw/markdown-to-sqlite) | Python | SQLite | Si | SQL (Datasette) | No | Mantenido |
| [frontmatter CLI](https://github.com/rythoris/frontmatter) | Go | stdout | Extrae | jq pipe | No | Minimal |
| [dirdocs](https://github.com/graves/dirdocs) | Rust | .dirdocs.nu | Genera via LLM | Nushell | No | Experimental |

**MarkdownDB** es el mas completo: indexa frontmatter + tags + backlinks en SQLite, tiene watch mode, y CLI (`npx mddb ./docs`). Pero no soporta reglas per-directory ni derivacion de estado jerarquico.

**markdown-to-sqlite** es minimalista (Simon Willison / Datasette ecosystem). Convierte markdown a SQLite y deja que el usuario haga queries SQL. Sin logica de negocio.

### B. MCP Servers (integracion AI, 2025-2026)

Servidores Model Context Protocol que permiten a AI assistants consultar knowledge bases markdown. **MCP adoption crecio 340% en 2025, con 500+ servers en registros publicos.**

| Herramienta | Backend | Capacidad | Limitacion vs docrules |
|-------------|---------|-----------|------------------------|
| [markdown-frontmatter-mcp](https://github.com/caffeinatedwes/markdown-frontmatter-mcp) | In-memory | Query por tags + dates, para Obsidian | Sin reglas, flat |
| [frontmatter-mcp (DuckDB)](https://github.com/kzmshx/frontmatter-mcp) | DuckDB | SQL sobre frontmatter + semantic search | Sin herencia por dir |
| [Knowledge Base MCP](https://lobehub.com/mcp/cwente25-knowledgebasemcp) | Files | CRUD + categorias, metadata por nota | Flat structure, sin jerarquia |
| [Library MCP](https://github.com/lethain/library-mcp) | Files | Folders + tags, operaciones basicas | Sin reglas, sin validacion |

**frontmatter-mcp con DuckDB** es interesante: trata frontmatter como tabla SQL via DuckDB. Soporta `query_inspect` para descubrir schema y `batch_update` para edicion masiva. Pero no tiene reglas de validacion ni herencia por directorio.

**Tendencia**: Los MCP servers representan la direccion 2025-2026 para integracion AI + knowledge bases. Un MCP server embebido en el binario CLI (`[app] serve`) seria diferenciador: la mayoria de MCP servers son TypeScript.

### C. PKM CLI / Zettelkasten

Herramientas de Personal Knowledge Management orientadas a terminal.

| Herramienta | Lenguaje | Enfoque | Frontmatter |
|-------------|----------|---------|-------------|
| [zk](https://github.com/sirupsen/zk) | Ruby | Zettelkasten CLI con sqlite + fzf | No |
| [qmd](https://github.com/tobi/qmd) | Node.js | Hybrid search (BM25 + vector + LLM rerank) | Partial |
| [markdown_query](https://github.com/ssosik/markdown_query) | - | Xapian indexer, NLP queries | Partial |
| Obsidian Dataview | JS (plugin) | Query language sobre vault | Si |

**Obsidian Dataview** es el referente conceptual mas cercano: permite queries sobre frontmatter con un lenguaje propio (`TABLE`, `LIST`, `TASK`). Pero requiere Obsidian GUI, no tiene CLI, y no soporta reglas per-directory.

**qmd** (by Tobi Lutke, CEO Shopify) es ambicioso: combina BM25 + semantic search + LLM reranking. Pero requiere ~2GB de modelos locales y esta orientado a busqueda, no a workflow/validacion.

### D. Per-Directory Config Inheritance (patron)

Herramientas que implementan el patron de configuracion por directorio con herencia.

| Modelo | Direccion | Mecanismo | Root stop |
|--------|-----------|-----------|-----------|
| `.htaccess` (Apache) | **parent→child** (hacia abajo) | Directivas heredan a subdirs, child override | ServerRoot |
| `.editorconfig` | **child→parent** (hacia arriba) | Busca config subiendo, merge | `root = true` |
| `.eslintrc` | **child→parent** (hacia arriba) | Cascade config subiendo, merge | `root: true` |
| `.gitattributes` | **parent→child** (hacia abajo) | Pattern matching por directorio | `.git` root |
| Alfresco folder rules | **parent→child** (hacia abajo) | Reglas por carpeta, herencia, child no edita parent | Admin config |

**docrules sigue el modelo `.htaccess` / `.gitattributes`**: las reglas se definen en el directorio padre y se propagan hacia abajo. Cada subdirectorio puede agregar reglas o hacer override, pero el flujo primario es top-down.

Esto contrasta con `.editorconfig`/`.eslintrc` donde la busqueda es bottom-up (el archivo pregunta "que reglas me aplican" subiendo por el arbol). Ambos modelos usan `root: true` como stop condition, pero la semantica es opuesta.

### E. Resume/Portfolio Generators

Herramientas que generan presentacion profesional desde datos estructurados.

| Herramienta | Enfoque | Limitacion |
|-------------|---------|------------|
| [JSON Resume](https://github.com/jsonresume/resume-cli) | Schema fijo, temas, PDF/HTML | Schema rigido, no extensible, datos manuales |
| [HackMyResume](https://github.com/hacksalot/HackMyResume) | Multi-formato (HTML, PDF, LaTeX, Word) | Abandonado (~2018), schema fijo |
| Portfolio estaticos (Astro, MkDocs, Jekyll) | Sites manuales con markdown | 100% manual, sin derivacion automatica |
| [vitae](https://pkg.mitchelloharawild.com/vitae/) | CV academico con R Markdown | Nicho academico, R dependency |

**Hallazgo**: Nadie genera portfolio o presentacion profesional a partir de **artefactos de trabajo reales** (PRDs, tasks, implementaciones). Todos requieren input manual o schema fijo. Este es un blue ocean complementario — no es el scope de docrules, pero el hecho de que docrules indexe trabajo real lo hace potencialmente valioso como fuente de datos para este tipo de herramientas.

### Hallazgo Clave

**Ninguna herramienta existente combina las 3 dimensiones:**

1. Indexacion de frontmatter → mddb, frontmatter-mcp lo hacen
2. Per-directory rules con herencia parent→child → **ninguna herramienta de docs lo hace**
3. Derivacion de estado jerarquico → ninguna (Dataview se acerca pero sin herencia)

Este es el espacio vacio que docrules llenaria.

---

## 5. Arquitectura Propuesta

### Componentes

```
[app] (binario Go)
├── parser/
│   ├── frontmatter.go     # YAML frontmatter parser
│   ├── inline.go          # Inline markdown metadata (legacy)
│   └── types.go           # Document struct unificado
├── rules/
│   ├── loader.go          # Lee .docrules con herencia parent→child
│   ├── merge.go           # Merge parent→child (override semantics)
│   └── validator.go       # Valida documento contra reglas
├── index/
│   ├── scanner.go         # Directory walker (respeta .gitignore)
│   ├── cache.go           # Cache JSON con mtime invalidation (futuro)
│   └── query.go           # Query engine
├── mcp/
│   ├── server.go          # MCP server (modelcontextprotocol/go-sdk)
│   └── tools.go           # MCP tools: query, validate, tree
└── cli/
    ├── query.go           # [app] query --type task --estado Pending
    ├── validate.go        # [app] validate [file|--all]
    ├── tree.go            # [app] tree [path]
    ├── stats.go           # [app] stats
    └── serve.go           # [app] serve (MCP server mode)
```

### CLI API

```bash
# Queries
[app] query --type task --estado Pending          # Tasks pendientes
[app] query --id T005                              # Task por ID
[app] query --tree epics --derive                  # Arbol con estados derivados

# Validacion
[app] validate docs/epics/E02/.../T005.md          # Validar un archivo
[app] validate --all                                # Validar todos

# Visualizacion
[app] tree docs/epics/                              # Arbol jerarquico
[app] stats                                         # Resumen por tipo/estado

# Output
[app] query --id T005 --format json                 # JSON para consumo programatico
[app] query --id T005 --format table                # Tabla para humanos

# MCP Server
[app] serve                                         # Iniciar MCP server (stdio)
[app] serve --transport sse --port 8080             # MCP server via SSE
```

### Ejemplo `.docrules`

```yaml
# docs/epics/.docrules
root: true

schema:
  levels:
    - name: epic
      pattern: "E[0-9][0-9]-*/"
      metadata_source: readme   # Lee README.md del directorio
      required_fields: [estado]
      valid_values:
        estado: [Activa, Completada, Diferida]

    - name: feature
      pattern: "F[0-9][0-9]-*/"
      metadata_source: readme
      derive_estado: true       # Derivado de Stories hijas

    - name: story
      pattern: "S[0-9][0-9][0-9]-*/"
      metadata_source: readme
      derive_estado: true       # Derivado de Tasks hijas

    - name: task
      pattern: "T[0-9][0-9][0-9]-*.md"
      metadata_source: frontmatter
      required_fields: [estado]
      optional_fields: [tipo, ejecutable_en]
      valid_values:
        estado: [Pending, Especificado, Completado, Obsoleto]
        tipo:
          - servicio-docker
          - modulo-sistema
          - operacion-sistema
          - lxc
          - vm
          - modulo-infraestructura
          - host-script
          - instance-script
          - documentation

derivation:
  # Como derivar estado de padres desde hijos
  rules:
    - when: all_children_completed
      then: Completado
    - when: any_child_in_progress
      then: En progreso
    - when: all_children_pending
      then: Specified
```

---

## 6. Stack Tecnico

### Decision: Go

**Go maximiza la rentabilidad profesional para las 3 audiencias simultaneamente.**

| Criterio | Go | Rust | TypeScript |
|----------|-----|------|------------|
| DevOps/SRE | **Lingua franca** (Docker, K8s, Terraform) | Respetado pero no estandar | No se toma en serio para infra |
| SWE general | Solido, pragmatismo | Impresiona, profundidad tecnica | Universal pero sin diferenciacion |
| AI/LLM | SDK MCP oficial (Go team + Anthropic) | SDK MCP oficial disponible | Default del ecosistema MCP |
| Tiempo al MVP | ~2-3 semanas | ~4-6 semanas (learning curve) | ~2 semanas |
| Binario | Estatico, zero-dep, cross-compile trivial | Estatico, zero-dep | Requiere Node.js runtime |
| Riesgo calidad | Bajo (Go es opinionated) | Alto (Rust mal escrito peor que Go bien escrito) | Medio |
| GitHub stars | Moderado (1K-10K tipico) | Alto (novelty factor) | Bajo-moderado (saturado) |
| Contribuciones | Facil (Go devs abundan) | Harder (barrier to entry) | Facil pero variable |

**Argumento decisivo**: Un proyecto Go bien terminado > un proyecto Rust a medias. El riesgo #1 de un portfolio piece es abandono. Go maximiza probabilidad de completar y mantener.

**Coherencia narrativa**: Un CLI en Go junto a un portfolio de IaC (OpenTofu, Docker, Proxmox) dice "hablo el mismo idioma que la infraestructura que gestiono". Rust seria disonante con la narrativa DevOps/Platform.

**MCP en Go como diferenciador**: La mayoria de MCP servers son TypeScript. Publicar uno en Go senala versatilidad y demuestra que se puede hacer tooling serio de AI en el lenguaje de infraestructura.

### Stack Especifico

| Componente | Libreria/Tool | Razon |
|------------|---------------|-------|
| CLI framework | `cobra` + `viper` | Estandar de facto (kubectl, gh, hugo) |
| YAML parsing | `gopkg.in/yaml.v3` | Maduro, battle-tested |
| Testing | `go test` + `testify` | Built-in + assertions legibles |
| MCP server | [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) | SDK oficial, Go team + Anthropic (ene 2026) |
| CI/CD | GitHub Actions + goreleaser | Releases automaticos multi-plataforma |
| Distribucion | `go install` + Homebrew tap | Maxima accesibilidad |

### Amplificadores de Impacto como Portfolio

1. **goreleaser** para binarios multi-plataforma en cada release
2. **Homebrew tap** para `brew install [app]`
3. **MCP server mode** (`[app] serve`) — diferenciador vs todos los CLI indexers
4. **Badge de cobertura** + CI/CD visible en README
5. **Dogfooding**: el repo del propio proyecto usa `.docrules` para validarse a si mismo

---

## 7. Scope

### Standalone desde Dia 1

| Aspecto | Project utility | Standalone |
|---------|-----------------|------------|
| Ubicacion | `.claude/tools/` en el repo | Repo propio, releases |
| Usuarios | Solo este homeserver | Cualquier proyecto con docs markdown |
| Testing | Ad-hoc | CI/CD, unit tests, integration tests |
| Distribucion | Copiar binario | go install, releases GitHub, brew |
| Documentacion | Inline en README | Docs site, ejemplos, tutoriales |

**Decision**: Standalone desde el inicio. El costo incremental es bajo (go.mod propio, tests) y el beneficio es alto (reutilizable, contribuible, publishable).

### Potencial Open Source

- **Nombre CLI**: TBD (ver Decisiones Abiertas)
- **Nombre archivo reglas**: `.docrules` (definitivo)
- **Nicho**: Equipos que usan markdown-as-documentation con workflows definidos
- **Diferenciador**: Per-directory rules con herencia parent→child + MCP server nativo

---

## 8. Que Migra de Skills/Hooks a la App

| Logica actual | Donde vive hoy | Donde viviria |
|---------------|-----------------|---------------|
| Buscar Task por ID | 6 skills (find+grep) | `[app] query --id T005` |
| Validar tipo de Task | 6 skills (grep inline) | `[app] validate --file T005.md` |
| Validar estado valido | Write hook (hardcoded) | `.docrules` valid_values |
| Derivar estado Story | `/roadmap view` (inline) | `[app] tree --derive` |
| Listar PRDs por estado | `/pendings` (grep) | `[app] query --tree prd --estado X` |
| Arbol completo | `/roadmap view` (rebuild) | `[app] tree` |
| Agregar nuevo tipo | Editar hook + N skills | Editar `.docrules` |

**Reduccion estimada**: ~60% del codigo de logica en skills se elimina, reemplazado por llamadas al CLI.

---

## 9. Decisiones

### Cerradas

| Decision | Resultado | Razon |
|----------|-----------|-------|
| **Lenguaje** | Go | Maximiza senal profesional para 3 audiencias; coherencia con portfolio IaC; menor riesgo de abandono |
| **Cache** | On-demand (sin cache) | YAGNI; ~200ms para 263 archivos es suficiente; agregar cache si se vuelve lento |
| **Formato .docrules** | YAML | Coherencia con frontmatter de los documentos que indexa |
| **Modelo herencia** | parent→child (.htaccess) | Las reglas se definen en Epic/Feature y se propagan a Stories/Tasks |
| **Scope** | Standalone desde dia 1 | Costo incremental bajo, beneficio alto (publicable, contribuible) |

### Abiertas

| Decision | Estado | Notas |
|----------|--------|-------|
| **Nombre del proyecto** | TBD | `.docrules` funciona como nombre de archivo de reglas, pero no como nombre de CLI/app — demasiado generico. Criterios: memorable, googleable, sugiera docs+estructura. Candidatos pendientes de evaluar en futura iteracion. |

---

## 10. Next Steps

Este documento es research puro. No crea compromisos de implementacion.

Opciones para avanzar (con tech stack Go ya decidido):

1. **Epic en el framework**: E03 dedicado al desarrollo como herramienta del ecosistema
2. **Repo standalone**: Crear repo separado, Go module, desarrollar como side project
3. **MVP scoped**: Version minima (~400 lineas Go) que cubra solo `docs/epics/` como prueba de concepto
4. **MCP-first**: Implementar como MCP server primero (integracion nativa con Claude Code, maximo impacto demostrativo)

---

## Referencias

### Competidores Directos
- [MarkdownDB (mddb)](https://github.com/datopian/markdowndb)
- [markdown-to-sqlite](https://github.com/simonw/markdown-to-sqlite)
- [frontmatter CLI](https://github.com/rythoris/frontmatter)
- [dirdocs](https://github.com/graves/dirdocs)

### MCP Servers
- [markdown-frontmatter-mcp](https://github.com/caffeinatedwes/markdown-frontmatter-mcp)
- [frontmatter-mcp (DuckDB)](https://github.com/kzmshx/frontmatter-mcp)
- [Knowledge Base MCP](https://lobehub.com/mcp/cwente25-knowledgebasemcp)
- [Library MCP](https://github.com/lethain/library-mcp)

### PKM / Search
- [zk (Zettelkasten CLI)](https://github.com/sirupsen/zk)
- [qmd (hybrid search)](https://github.com/tobi/qmd)
- [Obsidian Dataview](https://blacksmithgu.github.io/obsidian-dataview/)

### Resume/Portfolio
- [JSON Resume CLI](https://github.com/jsonresume/resume-cli)
- [HackMyResume](https://github.com/hacksalot/HackMyResume)

### Per-Directory Config Patterns
- [Apache .htaccess](https://httpd.apache.org/docs/current/howto/htaccess.html)
- [EditorConfig spec](https://editorconfig.org/)
- [Alfresco folder rules](https://docs.alfresco.com/content-services/6.1/using/content/rules/)

### Tech Stack
- [Go MCP SDK oficial](https://github.com/modelcontextprotocol/go-sdk) (Go team + Anthropic, ene 2026)
- [Rust MCP SDK oficial](https://github.com/modelcontextprotocol/rust-sdk)
- [Rust vs Go 2025 - JetBrains](https://blog.jetbrains.com/rust/2025/06/12/rust-vs-go/)
- [Building CLIs 2025: Node.js vs Go vs Rust](https://medium.com/@no-non-sense-guy/building-great-clis-in-2025-node-js-vs-go-vs-rust-e8e4bf7ee10e)
