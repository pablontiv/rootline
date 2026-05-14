---
titulo: Reddit Communities Launch Checklist
tipo: research
estado: borrador
fecha: 2026-03-26
---

# Reddit Communities Launch Checklist

Checklist de requisitos y tareas para publicar Rootline en cada subreddit recomendado.

## Requisitos Globales (aplican a TODOS los subs)

- [ ] **Cuenta con 30+ días de antigüedad** (idealmente 60+)
- [ ] **200–500+ karma combinado** antes de postear en subs estrictos
- [ ] **Regla 90/10**: 90% de actividad debe ser participación genuina, 10% autopromoción
- [ ] **10–20 comentarios no-promocionales** en cada sub objetivo antes de publicar
- [ ] **No usar link shorteners** (Bitly, Dub.co) — Reddit los marca como spam
- [ ] **No crosspostear contenido idéntico** — personalizar cada post
- [ ] **Escalonar publicaciones**: máximo 1–2 posts por día, distribuir en 2–3 semanas
- [ ] **Preparar assets**: GIFs de terminal (asciinema), screenshots de output, repo con README pulido
- [ ] **Estar disponible 30–60 min post-publicación** para responder comentarios

---

## Tier 1 — Alta Relevancia

### r/golang (~280k members)

**Reglas conocidas:**
- Gobernado por el [Go Community Code of Conduct](https://go.dev/conduct)
- Posts de proyectos son bienvenidos (flair "Show and Tell" o "Project")
- Tono estrictamente técnico — no lenguaje de marketing

**Checklist:**
- [ ] Verificar reglas actuales en sidebar: https://www.reddit.com/r/golang/about/rules
- [ ] Usar flair apropiado (probablemente "Show and Tell")
- [ ] Preparar post técnico: arquitectura Go, decisiones de diseño (cobra, expr-lang, merge por tipos YAML)
- [ ] Incluir code snippets relevantes
- [ ] Título sugerido: *"I built a file-based database engine in Go that treats the filesystem as a queryable database"*
- [ ] NO mencionar pricing/comercial — es 100% OSS
- [ ] Tener 10+ comentarios útiles previos en r/golang

### r/commandline (~350k members)

**Reglas conocidas:**
- Comunidad de CLI lovers — valoran UX de terminal
- Reglas específicas no indexadas públicamente

**Checklist:**
- [ ] Verificar reglas actuales en sidebar: https://www.reddit.com/r/commandline/about/rules
- [ ] Preparar GIF/asciinema demo mostrando: `rootline query`, `rootline tree`, `rootline validate`
- [ ] Post tipo text con demo embebido
- [ ] Título sugerido: *"rootline — treat your filesystem as a queryable database from the terminal"*
- [ ] Mostrar pipeline completo en acción
- [ ] Tener 10+ comentarios previos en el sub

### r/selfhosted (~350k members)

**Reglas conocidas:**
- **Flair obligatorio** en cada post
- **Solo software** — no hardware (excepto miércoles)
- **No ventas directas** — pero sí release notices de software
- **Blog posts**: NO link posts directos — deben ser text posts con contexto + link en body
- **No drama**
- Mods tienen última palabra sobre si la herramienta encaja

**Checklist:**
- [ ] Verificar reglas completas: https://www.reddit.com/r/selfhosted/about/rules
- [ ] Seleccionar flair apropiado (probablemente "New Software" o similar)
- [ ] Crear text post (NO link post) con descripción + link al repo en el body
- [ ] Incluir: qué es, por qué es relevante, cómo ayuda — no solo un link
- [ ] Ángulo: *"No database server needed — your filesystem IS the database"*
- [ ] Enfatizar: zero dependencies, local-first, plain Markdown/YAML, no vendor lock-in
- [ ] Tener 10+ comentarios previos en r/selfhosted

### r/ObsidianMD (~250k members)

**Reglas conocidas:**
- Posts de tipo "[Showcase]" son comunes para herramientas
- Comunidad estricta con self-promotion — verificar reglas antes
- Wiki-links (`[[link]]`) y YAML frontmatter son conceptos nativos aquí

**Checklist:**
- [ ] Verificar reglas actuales en sidebar: https://www.reddit.com/r/ObsidianMD/about/rules
- [ ] Verificar si requieren flair tipo "[Showcase]"
- [ ] NO compartir link directamente — describir la solución al problema primero
- [ ] Ángulo: *"Validate and query your Obsidian vault's frontmatter from the CLI"*
- [ ] Mostrar: detección de links rotos, validación de frontmatter, dependency graphs
- [ ] Preparar ejemplo concreto con un vault de Obsidian
- [ ] Participar activamente respondiendo preguntas sobre frontmatter/YAML antes de postear
- [ ] Tener 15+ interacciones previas (este sub es más estricto)

### r/PKMS (~57k members)

**Reglas conocidas:**
- Posts de self-promotion existen (13 registrados en análisis recientes)
- Audiencia muy técnica en knowledge management

**Checklist:**
- [ ] Verificar reglas actuales: https://www.reddit.com/r/PKMS/about/rules
- [ ] Verificar si hay weekly thread para herramientas
- [ ] Ángulo: *"Schema inheritance and constraint validation for large document collections"*
- [ ] Enfatizar: `.stem` files, validación, estructura forzada
- [ ] Responder preguntas sobre PKM workflow antes de postear
- [ ] Tener 10+ interacciones previas

---

## Tier 2 — Relevancia Media

### r/LocalLLaMA (~662k members)

**Reglas conocidas:**
- **Stay on topic**: posts deben ser sobre LLMs
- **Regla del 10%**: self-promotion max 10% del contenido
- **No low-effort posts**: buscar respuestas existentes antes de preguntar
- **Links directos, no clickbait**
- Seguir Reddit Content Policy

**Checklist:**
- [ ] Verificar reglas: https://www.reddit.com/r/LocalLLaMA/about/rules
- [ ] Ángulo MCP: *"9 MCP tools for structured documentation that any LLM can use"*
- [ ] Mostrar workflow: AI agent usando Rootline MCP tools para validar/consultar docs
- [ ] Link directo al repo — no clickbait
- [ ] Participar en discusiones sobre MCP/tool-use antes de postear
- [ ] Tener 10+ comentarios previos en el sub

### r/ClaudeAI (~612k members)

**Reglas conocidas:**
- Subreddit NO operado por Anthropic
- Discusiones basadas en evidencia — respaldar claims con ejemplos
- "Search first" — verificar que no existe discusión previa
- Foco profesional: business applications, coding workflows

**Checklist:**
- [ ] Verificar reglas: https://www.reddit.com/r/ClaudeAI/about/rules
- [ ] Buscar posts previos sobre MCP servers o herramientas similares
- [ ] Ángulo: *"I built an MCP server that lets Claude validate and query structured docs"*
- [ ] Incluir ejemplo concreto de workflow con Claude + Rootline
- [ ] Proporcionar contexto completo (use case, setup, resultados)
- [ ] Participar respondiendo preguntas sobre Claude/MCP antes
- [ ] Tener 10+ interacciones previas

### r/technicalwriting (~60k members)

**Reglas conocidas:**
- Comunidad de escritores técnicos — no desarrolladores
- Reglas específicas no indexadas — verificar sidebar

**Checklist:**
- [ ] Verificar reglas: https://www.reddit.com/r/technicalwriting/about/rules
- [ ] Adaptar lenguaje: hablar de docs, no de código
- [ ] Ángulo: *"Enforce frontmatter consistency across 500+ Markdown files automatically"*
- [ ] Mostrar `validate --all` detectando errores reales
- [ ] Enfatizar que no requiere programar — schemas en YAML
- [ ] Mencionar `rootline fix` como fix automático
- [ ] Tener 10+ interacciones previas respondiendo sobre docs-as-code

### r/devtools (~30k members)

**Checklist:**
- [ ] Verificar reglas: https://www.reddit.com/r/devtools/about/rules
- [ ] Post de showcase: problema → solución → pipeline completo
- [ ] Mencionar MCP integration como diferenciador
- [ ] Tener 5+ interacciones previas (sub más pequeño, menos estricto)

### r/coolgithubprojects (~64k members)

**Reglas conocidas:**
- **Diseñado para compartir proyectos** — self-promotion es el propósito
- Muy amigable con OSS

**Checklist:**
- [ ] Verificar reglas: https://www.reddit.com/r/coolgithubprojects/about/rules
- [ ] Descripción breve + link directo al GitHub repo
- [ ] Título claro y descriptivo
- [ ] Asegurar que el README del repo esté pulido antes de postear
- [ ] **Mínimo karma/historial requerido** — puede ser de los primeros posts

### r/opensource (~210k members)

**Reglas conocidas:**
- Self-promotion **altamente limitada** (rating 🟡)
- Max ~10% del contenido total en el sub

**Checklist:**
- [ ] Verificar reglas: https://www.reddit.com/r/opensource/about/rules
- [ ] Contar la historia: por qué lo construiste, qué problema resuelve, filosofía
- [ ] No solo tirar link — dar contexto extenso
- [ ] Participar activamente en discusiones OSS antes
- [ ] Tener 15+ interacciones previas (sub más estricto con self-promo)

---

## Tier 3 — Nicho

### r/Zettelkasten (~40k members)

**Checklist:**
- [ ] Verificar reglas: https://www.reddit.com/r/Zettelkasten/about/rules
- [ ] Ángulo: dependency graphs, wiki-link resolution, schema enforcement
- [ ] Mostrar cómo `.stem` schemas pueden forzar convenciones Zettelkasten
- [ ] Participar en discusiones sobre estructura de notas antes
- [ ] Tener 5+ interacciones previas

### r/plaintext (~10k members)

**Checklist:**
- [ ] Verificar reglas: https://www.reddit.com/r/plaintext/about/rules
- [ ] Ángulo: plain-text philosophy, Markdown + YAML, sin formatos propietarios
- [ ] Enfatizar que Rootline NO cambia el formato de archivos
- [ ] Tener 5+ interacciones previas

### r/Zettelkasten, r/KnowledgeManagement (~15k)

**Checklist:**
- [ ] Verificar reglas de cada uno individualmente
- [ ] Ángulo: infraestructura para knowledge engineering organizacional
- [ ] Mostrar schema enforcement, derivation, aggregation
- [ ] Tener 5+ interacciones previas en cada uno

### r/sysadmin (~800k) / r/devops (~300k)

**Checklist:**
- [ ] Verificar reglas (subs grandes = reglas estrictas)
- [ ] Ángulo: runbooks, incident docs, validación en CI pipelines
- [ ] Mostrar pre-commit hooks + `validate --all` en pipelines
- [ ] **Requiere karma alto y participación extensa** — estos subs son muy estrictos
- [ ] Tener 20+ interacciones previas

---

## Calendario Sugerido de Publicación

| Semana | Día | Subreddit | Prioridad |
|--------|-----|-----------|-----------|
| 1 | Mar | r/coolgithubprojects | Bajo riesgo, warm-up |
| 1 | Jue | r/golang | Alta |
| 2 | Mar | r/commandline | Alta |
| 2 | Jue | r/ObsidianMD | Alta |
| 3 | Mar | r/LocalLLaMA | Media |
| 3 | Jue | r/selfhosted | Alta |
| 4 | Mar | r/ClaudeAI | Media |
| 4 | Jue | r/technicalwriting | Media |
| 5 | Mar | r/PKMS | Alta |
| 5 | Jue | r/opensource | Media |
| 6+ | — | Nicho (Zettelkasten, plaintext, etc.) | Baja |

**Mejor horario**: Martes a Jueves, 9–11 AM US Eastern (14–16 UTC).

---

## Fase de Preparación (antes de publicar)

- [ ] Construir karma a 500+ (r/AskReddit, r/golang, r/programming — 2 semanas)
- [ ] Tener 10–20 comentarios útiles en cada sub objetivo
- [ ] README del repo pulido con badges, screenshots, quick start
- [ ] Grabar 2–3 demos asciinema (query, validate, tree, graph)
- [ ] Preparar drafts personalizados para cada sub
- [ ] Revisar reglas de CADA sub el día antes de publicar (pueden cambiar)

## Fuentes

- [SubredditSignals — 11 Proven Subreddits to Promote Tech 2026](https://www.subredditsignals.com/blog/best-subreddits-to-promote-a-tech-product-in-2026-rules-real-examples-and-outreach-tips-that-don-t-get-you-banned)
- [Market Clarity — Top 25 Subreddits Where Self-Promotion Is Allowed](https://mktclarity.com/blogs/news/list-subreddits-promotion)
- [KarmaGuy — 10 Best Subreddits to Build Karma Quickly](https://karmaguy.io/en/blog/best-subreddits-for-karma)
- [Prowlo — Reddit Marketing for DevTools](https://prowlo.com/blog/reddit-marketing-devtools)
- [Reddit Help — What constitutes spam](https://support.reddithelp.com/hc/en-us/articles/360043504051-What-constitutes-spam-Am-I-a-spammer)
- [Reddit Help — Responsible Builder Policy](https://support.reddithelp.com/hc/en-us/articles/42728983564564-Responsible-Builder-Policy)
