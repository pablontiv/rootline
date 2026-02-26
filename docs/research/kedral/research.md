# Kedral — Research

**Fecha**: 2026-02-25
**Tipo**: Research
**Contexto**: Herramienta CLI en Go que mantiene una Known Error Database cross-proyecto. Responde en tiempo real durante sesiones de Claude Code, y crea KEs automaticamente cuando un error se confirma como recurrente. Apoyada en Backscroll (descubrimiento), Rootline (estructura/validacion) y SQLite FTS5 (matching). Sin LLM embebida — Claude Code ya es el LLM.

---

## 1. Problema

El conocimiento operacional descubierto durante sesiones de Claude Code se pierde o queda enterrado en archivos MEMORY.md no estructurados. Analisis de sesiones cross-proyecto revela ~15 known errors documentados informalmente solo en homeserver-automation, con patrones recurrentes que incluyen intentos fallidos, workarounds y reglas derivadas — informacion de alto valor que ningun ITSM captura formalmente.

### Evidencia del problema

| Proyecto | Known errors informales en MEMORY.md | Lineas |
|----------|--------------------------------------|--------|
| homeserver-automation | ~15 (bpg/proxmox, cloud-init, Velero, nested virt, PBS, SSH, etc.) | 132 |
| vdc | 5 (Critical Errors to Avoid) | 30 |
| incubadora | 6 inconsistencias alta severidad | 27 |
| rootline | 1 (coverage gate) | 17 |

### Limitaciones del modelo actual (MEMORY.md como KEDB informal)

| Limitacion | Impacto |
|-----------|---------|
| Sin estructura formal | No se puede filtrar por estado, severidad, componente |
| Sin validacion | Campos faltantes, formatos inconsistentes |
| Sin queryability | `grep` es la unica opcion; no hay `WHERE severidad == 'critical'` |
| Sin lifecycle | KEs nunca se retiran, revisan ni actualizan sistematicamente |
| Sin cross-proyecto | Cada MEMORY.md es isla; un KE de bpg/proxmox no se surfacea en otro proyecto |
| Sin trazabilidad | No hay link a las sesiones donde se descubrio/resolvio el error |
| Creacion manual | Depende de que el usuario se detenga a documentar (KCS violation) |

## 2. Estado del Arte

### 2.1 KEDB Tradicional (ITIL/ITSM)

| Herramienta | Modelo | Limitacion clave |
|-------------|--------|------------------|
| ServiceNow | Known Error como estado de Problem record (no tabla separada) | SaaS pesado, 15+ campos obligatorios |
| Jira SM | Problems en Jira + Confluence para KB | Hibrido fragil, linking manual |
| BMC Helix | Problem Management con KCS integrado | Enterprise-only |
| Freshservice | Problem module + Knowledge Base separada | Promocion manual de workarounds a KB |

**Hallazgo clave**: ServiceNow ya no usa KEDB como tabla separada. "Known Error" es un estado del ciclo de vida del Problem, no una entidad independiente. La industria converge hacia este modelo.

**Dato**: >60% de issues en ServiceNow ya tenian solucion pero no estaba capturada de forma encontrable (ServiceNow KCS Case Study).

### 2.2 SRE Moderno (Post-ITIL)

**Google SRE NO mantiene KEDB.** El corpus de postmortems searchable ES la base de conocimiento. Practican:

1. Postmortem como unidad de conocimiento (Google Docs/wiki, no database)
2. Templates estandarizados para analisis cruzado
3. Newsletter mensual de postmortems interesantes
4. Postmortem working group cross-equipo

**Rootly, incident.io, FireHydrant**: Ninguno mantiene KEDB separada. La retrospectiva/postmortem searchable es el modelo dominante. Todas usan event-sourcing: timeline de eventos → LLM genera draft de postmortem → base de conocimiento.

**Dato**: Postmortems auto-generados por LLM: 10-15 min de revision vs 90 min de escritura manual (incident.io benchmark 2025).

### 2.3 Patron Event-Sourced Knowledge

El patron mas efectivo identificado en la industria:

```
Eventos (incidentes/sesiones) → Pattern detection → Knowledge entry (automatico)
```

Implementaciones:

| Herramienta | Mecanismo | Resultado |
|-------------|-----------|-----------|
| Rootly | Timeline de eventos → LLM draft postmortem | Auto-sync a Confluence |
| Jeli (PagerDuty) | Slack history → Narrative Builder → postmortem | Knowledge elicitation metodologia "Howie" |
| Shoreline.io | Known error entry = automation ejecutable (Op) | MTTR -75%, 50% auto-remediacion |
| AIOps (Datadog, Splunk) | ML clustering de alertas → known error candidates | 98% reduccion de ruido como precondicion |

### 2.4 Metricas de Impacto

| Metrica | Fuente | Dato |
|---------|--------|------|
| Mejora en time-to-resolution con KCS | Consortium for Service Innovation | 50-60% |
| Reduccion MTTR con KE ejecutable | Shoreline.io | >75% |
| Reduccion MTTR con ownership clara en catalogo | FireHydrant (50k+ incidents) | 36% |
| Mejora en doc quality con AI | DORA 2024 (39k respondents) | 25% adopcion AI → 7.5% boost doc quality |

### 2.5 Por Que Fallan los KEDBs

De la investigacion cross-fuente, los 8 failure modes confirmados:

| # | Failure Mode | Causa | Mitigacion en KEDB tool |
|---|-------------|-------|-------------------------|
| 1 | Friccion de creacion (blank page) | Formularios de 15+ campos | Claude Code escribe el KE (ya tiene el contexto completo) |
| 2 | Nadie lo encuentra (discovery) | Modulo separado, search pobre | Hook intercepta prompt y responde automaticamente |
| 3 | Staleness (confianza erosionada) | Sin fecha de revision, sin owner | Auto-deteccion de staleness, `retire` command |
| 4 | Scope creep (knowledge dump) | FAQs/how-tos mezclados con KEs | Schema `.stem` fuerza estructura KEDB-only |
| 5 | Sin ownership | Nadie conduce resolucion | `owner: required` en schema |
| 6 | Desconexion con cambios | Fix deployed pero KE sigue activo | Deteccion de componente removido/actualizado |
| 7 | Complejidad del tool | ITSM pesado | CLI ligera, ~10MB, pure Go, zero deps |
| 8 | Rechazo cultural ("not my job") | Proceso separado del trabajo | KEs creados automaticamente al confirmar recurrencia |

### 2.6 Cross-Repo Knowledge Linking

**No esta resuelto a nivel de filesystem.** No existe mecanismo nativo de git para wiki-links cross-repo. Backstage lo resuelve en build-time (TechDocs). Obsidian rechaza cross-vault linking por diseno.

**Implicacion**: La integracion cross-proyecto es via tooling (MCP, CLI, hooks), no via filesystem. Submodules y symlinks no resuelven el problema semantico.

### 2.7 Campos No-Estandar de Alto Valor

Del analisis de MEMORY.md real, se identificaron campos que ningun ITSM/KEDB captura formalmente:

| Campo | Evidencia | Valor |
|-------|-----------|-------|
| `intentos_fallidos` | 80% de KEs en homeserver documentan fixes que NO funcionaron | Evita repetir caminos muertos — mas valioso que el workaround |
| `regla_derivada` | "NUNCA skip hooks", "NUNCA Kopia standalone" | Evolucion natural KE → politica organizacional |
| `tipo` categorizado | bug_externo, error_procedimiento, limitacion_plataforma | Categorizacion DevOps no-ITIL |
| `sesiones_origen` | Trazabilidad a raw session data | Auditable — link directo al incidente via Backscroll |
| `recurrencia` | Conteo automatico de apariciones | Priorizacion basada en datos |

### 2.8 Error Matching Sin LLM — Estado del Arte

**Nota arquitectural**: En la arquitectura refinada, FTS5/BM25 es responsabilidad de Backscroll (Tier 2 search). Kedral no mantiene index FTS5 propio — orquesta consultas a Rootline (Tier 1, structured) y Backscroll (Tier 2, full-text).

#### Sentry: fingerprinting determinístico

Sentry agrupa millones de errores con un pipeline **100% deterministico** (antes de su capa AI):

1. Normaliza stack traces (strip generics, params, line numbers)
2. Hashea la tupla `(module, function, context_line)` de frames in-app
3. SHA del hash = fingerprint del grupo

**Dato**: Cuando Sentry agrego su capa AI (embeddings), solo redujo grupos nuevos en 40%. El 60% restante ya estaba bien agrupado con el pipeline deterministico puro.

#### BM25/FTS5 para error matching tecnico

**Dato**: BM25 alcanza nDCG@10 de ~43% en BEIR benchmark general. Pero para **tokens tecnicos exactos** (nombres de excepciones, funciones, error codes), BM25 **supera** a dense retrieval porque los embeddings tratan tokens inusuales como ruido.

**Dato**: Character n-gram TF-IDF (sin ML) alcanza 83.5% macro / 92.4% micro F1 en clasificacion de eventos (ACL 2020).

**Elasticsearch More Like This** es literalmente TF-IDF: extrae top-N terminos por score TF-IDF del documento seed, construye un `bool/should` query, y rankea con BM25. No hay ML.

#### Drain algorithm para template mining

Drain (2017, He et al.) extrae templates de logs en tiempo lineal O(n), streaming:

```
Input:  "connection refused to 192.168.1.1 port 5432"
        "connection refused to 10.0.0.5 port 5432"
Template: "connection refused to <*> port 5432"
```

Mejor accuracy que todos los otros parsers en benchmark de 16 datasets. Dos hiperparametros: profundidad del arbol y threshold de similaridad. Implementable en ~200 lineas de Go.

#### Patron Zendesk: busqueda en dos tiers

Zendesk/Intercom buscan primero en KB curada, luego en tickets raw:

```
Tier 1: KB article match → respuesta directa (confianza alta)
Tier 2: Ticket history match → "otros reportaron algo similar" (confianza media)
```

Este patron de dos tiers con niveles de confianza distintos es el modelo para `kedral search`: Tier 1 = KEDB (Rootline), Tier 2 = Backscroll (sesiones).

#### Latencia de inyeccion pre-respuesta

El budget de latencia para inyeccion de contexto antes de respuesta AI es <200ms (tolerancia de usuario para "typing indicator"). FTS5 query sobre 50 KEs + 270 sessions < 10ms. Sobra margen.

### 2.9 Patron de Confirmacion por Recurrencia

No se encontro un nombre formal en la literatura de KM, pero el patron aparece en multiples dominios:

| Dominio | Implementacion | Threshold |
|---------|---------------|-----------|
| Spam detection | Email marcado como spam solo despues de N reportes | 2-5 reportes |
| Sentry | Nuevo issue creado al primer evento, pero alertas solo en N+ eventos | Configurable |
| Wikipedia | Stub → Article → Featured (promocion por calidad + uso) | Criterios cualitativos |
| PagerDuty IAG | Agrupa alertas despues de 5-10 merges manuales observados | 5-10 observaciones |

**Aplicacion en KEDB**: No crear KE en la primera ocurrencia. Crear KE cuando:
1. El mismo error aparece en 2+ sesiones distintas (recurrencia confirmada)
2. El workaround se confirma (funciono ambas veces)

Esto elimina falsos positivos de raiz — el KE nace como `active`, no `draft`.

### 2.10 El KE como Bridge Entity

#### Cross-domain Bridge Entity patterns

De la investigacion cross-dominio (6 dominios), el registro KE se mapea a patrones de bridge entity bien establecidos:

| Domain | Raw Events | Bridge Entity | Curated Knowledge |
|--------|-----------|---------------|-------------------|
| Zettelkasten | Source material | Literature Note | Permanent Note |
| ITIL v4 | Incident records | Known Error Record | Knowledge Article in SKMS |
| KCS | Support case | Article (created from + linked to case) | Same article (after validation) |
| Incident pipelines | Slack messages, alerts | Postmortem (auto-generated) | Action items, knowledge articles |
| AI Agent Memory (Zep/Graphiti) | Conversation messages | Entity-edge with temporal validity | Structured knowledge graph |
| **Este ecosistema** | **Backscroll (sessions + plans)** | **KE record (Kedral)** | **Rootline (filesystem DB)** |

Tres patrones arquitecturales:

1. **Collapsed Bridge** (KCS): el article ES el bridge. Mas simple pero pierde la distincion entre observacion y conocimiento.
2. **Explicit Bridge Entity** (ITIL, Zettelkasten): entidad intermedia dedicada con referencias bidireccionales. Mayor trazabilidad. **Este es el patron que KEDB implementa.**
3. **Extraction Pipeline** (AI Agent Memory): proceso automatizado que extrae conocimiento estructurado del event stream. El bridge es el mecanismo de extraccion en si mismo.

#### Bi-temporal validity (patron de Zep/Graphiti)

Los registros KE tienen validez bi-temporal implicita:

- `fecha_creacion` = cuando lo aprendimos (T' ingestion time)
- `sesiones_origen[0]` date = cuando ocurrio por primera vez (T event time)
- `ultima_revision` = cuando se confirmo que sigue vigente
- `estado: resolved` = invalidacion temporal (el hecho ya no es verdadero)

Este patron aparece en Zep/Graphiti como "temporal knowledge graph" — cada edge tiene validez temporal, y los facts se invalidan explicitamente cuando dejan de ser verdaderos. En KEDB el ciclo de vida del `estado` cumple esta funcion sin necesidad de un grafo temporal explicito.

#### Industry validation: PagerDuty SRE Agent

PagerDuty construyo independientemente su SRE Agent con 3 tipos de memoria:

1. **Service-scoped observations** (lo que ha visto)
2. **Incident recollections** (separando signal de noise)
3. **Human-promoted playbooks** (fixes endorsados)

Clientes reportaron que la memoria era "make or break". Resultado: 50% faster incident resolution.

El mapeo al ecosistema Kedral es directo:
- Observations → Backscroll sessions (raw event stream)
- Recollections → KE records (bridge entity, signal curado)
- Playbooks → Rootline docs (conocimiento promovido y estructurado)

**Fuentes**:
- KCS Solve Loop: https://library.serviceinnovation.org/KCS/KCS_v6/KCS_v6_Practices_Guide/030/030
- ITIL Problem Management: https://wiki.en.it-processmaps.com/index.php/Problem_Management
- Zep/Graphiti temporal KG: https://arxiv.org/abs/2501.13956
- PagerDuty SRE Agent: https://www.pagerduty.com/blog/ai/we-built-an-sre-agent-with-memory-and-its-transforming-incident-response/

### 2.11 Daemon vs CLI — Evidencia de Sistemas Reales

#### Pain points universales de daemons

De la investigacion de Atuin, Raycast, Copilot, Sentry Relay, Datadog Agent, gopls, rust-analyzer:

| Problema | Ejemplos reales |
|----------|----------------|
| Memory leaks | gopls: 9-14GB en proyectos grandes. rust-analyzer: 2.5GB en proyecto minimo. Datadog: leak cronico en ECS. Todos tienen issues abiertos |
| Estado stale | Copilot: index silenciosamente inutil arriba de 750 archivos. Sentry Relay: reporta "healthy" mientras esta roto |
| Degradacion silenciosa | El peor modo de fallo — el usuario nunca sabe que el daemon dejo de funcionar bien |
| Tax de "siempre corriendo" | Raycast 120MB + Datadog 95MB + gopls 500MB+ = 700MB+ idle |
| "Restart it" es el fix universal | Todos los troubleshooting guides empiezan con "reiniciar el daemon" |

Budget de recursos tolerado por usuarios:

| Recurso | Aceptable | Quejas |
|---------|-----------|--------|
| RAM | <100MB | >1GB |
| CPU idle | <1% | >10% |
| Startup | <1s | >30s |

#### Sistemas que eligieron CLI sobre daemon

| Sistema | Modelo | Por que |
|---------|--------|---------|
| Atuin | CLI + daemon opcional | Daemon solo para lock contention. Search bypassa daemon |
| Renovate/Dependabot | CLI (cron) | Corre 1x/dia. No necesita estar vivo |
| SonarQube | CLI (CI-triggered) + server para estado | Analisis es CLI |

#### Decision: CLI on-demand + state file

Para ~50 KEs + Backscroll DB de ~45MB:

- **CLI on-demand**: ~12ms por query (5% del budget de 200ms del hook)
- **Daemon**: ~1ms por query pero +complejidad operacional
- Los ~11ms ahorrados no justifican systemd, PID management, health checks, y el riesgo de memory leaks
- State file (JSON) persiste recurrence counters y timestamps entre invocaciones

**Fuentes**:
- gopls memory: https://github.com/golang/go/issues/47855
- rust-analyzer memory: https://github.com/rust-lang/rust-analyzer/issues/20028
- Datadog memory leak: https://github.com/DataDog/datadog-agent/issues/29319
- Sentry Relay issues: https://github.com/getsentry/self-hosted/issues/3566
- Copilot 750-file limit: https://github.com/orgs/community/discussions/152490
- Atuin daemon: https://forum.atuin.sh/t/moving-atuin-to-a-daemon/78

## 3. Escenarios Aspiracionales con Datos

| Escenario | Dato | Fuente |
|-----------|------|--------|
| "Vi este error antes pero no encuentro donde" | 61% de devs gastan >30 min/dia buscando soluciones | SO 2024 Survey |
| "El fix estaba en un Slack thread hace 3 meses" | Fortune 500 pierden $31.5B/ano por KM failures | IDC |
| "Rompimos lo mismo dos veces" | Cloudflare: 2 outages en meses consecutivos, misma causa raiz (Nov+Dec 2025) | Cloudflare Blog |
| "Nuevo dev cayo en la misma trampa" | Onboarding: 3-9 meses promedio. Primeros 3 meses = valor negativo | DX Newsletter |
| "El workaround dejo de funcionar" | 68% de docs tecnicos no se actualiza en 6+ meses | Zoomin study |
| "Error se propaga cross-proyecto" | Log4Shell afecto 93% de enterprise cloud environments | Wiz/EY |
| "Claude sigue sugiriendo lo que no funciona" | GitHub issue #8209: "Claude prioriza memoria episodica sobre procedural" | anthropics/claude-code |

---

### "Vi este error antes pero no encuentro donde"

El desarrollador sabe que resolvio algo similar hace semanas, pero no recuerda en que sesion ni en que proyecto. Buscar manualmente en Backscroll o MEMORY.md toma 30+ minutos y frecuentemente falla.

Con KEDB, el hook intercepta el prompt, ejecuta `kedral search` en <10ms, y surfacea el KE con workaround validado antes de que Claude empiece a responder.

```
You: "tofu apply is showing encryption_key_fingerprint unknown after apply again"

[KEDB hook fires, <10ms]
> Known Error KE-0001 | medium | workaround available
>   bpg/proxmox encryption_key_fingerprint unknown after apply
>   Workaround: tofu untaint after the error. Resource IS created correctly.
>   DO NOT try: tofu taint + re-apply (recreates resource, causes downtime)
>   Recurrence: 3 | Last seen: 2026-02-20

Claude: "This is a known issue. Run `tofu untaint`..."
```

### "El fix estaba en un Slack thread hace 3 meses"

El conocimiento existe en algun raw event stream (Slack, sesion de Claude Code, ticket) pero no fue promovido a knowledge curado. La organizacion pierde $31.5B/ano (Fortune 500 aggregate) por este tipo de KM failure.

Con KEDB, el Tier 2 search busca en Backscroll sessions y surfacea snippets relevantes. Si el error se resuelve, el hook Stop crea automaticamente un KE — promoviendo el conocimiento del event stream al bridge entity.

```
You: "velero backup is failing with 'partially failed' on the NFS volumes"

[KEDB hook fires, <10ms]
> No formal KE, but previous sessions found:
>   [SESSION] amber-sleeping-panda (2025-11-18):
>     ...Velero PVC backup fails on NFS. Fix: use Restic instead of Kopia...
>   [SESSION] crystal-jumping-wolf (2025-12-03):
>     ...same NFS issue. Confirmed: Kopia has known bug with NFS locks...
>   If resolved, KE will be created automatically.

Claude: "Based on previous sessions, this is a known Kopia+NFS issue..."
```

### "Rompimos lo mismo dos veces"

Cloudflare tuvo 2 outages en meses consecutivos (Nov + Dec 2025) por la misma causa raiz. Sin un sistema que detecte recurrencia, cada incidente se trata como nuevo.

Con KEDB, el patron de confirmacion por recurrencia (S2.9) detecta automaticamente cuando un error aparece en 2+ sesiones. El KE se crea con trazabilidad a ambas ocurrencias, y la proxima vez que alguien enfrente el mismo error, el workaround aparece instantaneamente.

```
You: "the cert-manager renewal is stuck again, pods are in CrashLoopBackOff"

[KEDB hook fires, <10ms]
> Known Error KE-0012 | high | workaround available
>   cert-manager renewal stuck after cluster upgrade
>   Workaround: delete the challenge resources, NOT the certificate.
>   DO NOT try: kubectl delete certificate (triggers full re-issuance, 24h delay)
>   Recurrence: 4 | Last seen: 2026-02-10

Claude: "This is a recurring issue. Delete only the challenge resources..."
```

### "Nuevo dev cayo en la misma trampa"

Onboarding toma 3-9 meses promedio. En los primeros 3 meses, el nuevo dev tiene valor neto negativo porque tropieza con problemas ya resueltos por el equipo. Sin KEDB, cada dev nuevo repite los mismos errores.

Con KEDB, los KEs cross-proyecto actuan como memoria institucional. El nuevo dev recibe automaticamente los workarounds y los `intentos_fallidos` — evitando caminos muertos que otros ya exploraron.

```
You: "I'm setting up Proxmox cloud-init and the VM won't get an IP"

[KEDB hook fires, <10ms]
> Known Error KE-0003 | high | workaround available
>   cloud-init IP assignment fails with virtio NIC on Proxmox 8.x
>   Workaround: set net0 model to 'virtio' AND add cicustom with network-config v2.
>   DO NOT try: using model 'e1000' (works but 10x slower network)
>   DO NOT try: cloud-init without cicustom (Proxmox ignores ipconfig0 silently)
>   Recurrence: 5 | Last seen: 2026-02-18

Claude: "This is a well-documented issue. You need both virtio AND cicustom..."
```

### "El workaround dejo de funcionar"

68% de docs tecnicos no se actualiza en 6+ meses. Un workaround que funcionaba con la version anterior del provider/tool puede dejar de funcionar despues de un upgrade, y nadie actualiza la documentacion.

Con KEDB, `kedral retire` detecta KEs sin nueva recurrencia en 6+ meses y los marca para revision. Ademas, el campo `ultima_revision` se actualiza automaticamente cada vez que el KE matchea en una sesion, manteniendo la confianza temporal.

```
You: "the tofu untaint workaround for encryption_key_fingerprint isn't working anymore after provider upgrade to 0.68"

[KEDB hook fires, <10ms]
> Known Error KE-0001 | medium | workaround available
>   bpg/proxmox encryption_key_fingerprint unknown after apply
>   Workaround: tofu untaint after the error.
>   Recurrence: 3 | Last seen: 2026-02-20

[Session resolves: workaround changed after provider 0.68]
[Hook Stop fires]
> KE-0001 updated:
>   Workaround revised for provider >=0.68
>   ultima_revision: 2026-02-25
>   recurrencia: 4
```

### "Error se propaga cross-proyecto"

Log4Shell afecto 93% de enterprise cloud environments. Errores en dependencias compartidas (providers, librerias, plataformas) impactan multiples proyectos. Sin cross-proyecto linking, cada proyecto redescubre el mismo error independientemente.

Con KEDB, el campo `proyectos[]` y `componentes[]` permiten surfacear KEs cross-proyecto. Un KE descubierto en homeserver-automation se muestra automaticamente cuando el mismo componente causa problemas en otro proyecto.

```
You: "terraform plan shows drift on all pbsStorageContent resources in the vdc project"

[KEDB hook fires, <10ms]
> Known Error KE-0001 | medium | workaround available
>   bpg/proxmox encryption_key_fingerprint unknown after apply
>   Projects: homeserver, vdc
>   Workaround: tofu untaint after the error. Resource IS created correctly.
>   Note: Same root cause across projects — provider bug in bpg/proxmox
>   Recurrence: 5 | Last seen: 2026-02-22

Claude: "This is the same provider bug affecting homeserver. Run `tofu untaint`..."
```

### "Claude sigue sugiriendo lo que no funciona"

Sin `intentos_fallidos` estructurados, Claude Code puede sugerir fixes que ya se probaron y no funcionan. La memoria episodica (sessions) se pierde o no tiene suficiente peso frente al training data del modelo.

Con KEDB, los `intentos_fallidos` se inyectan como context negativo explicitamente, antes de que Claude genere su respuesta. El modelo recibe tanto el workaround correcto como los caminos muertos a evitar.

```
You: "nested virtualization isn't working on the Proxmox VM"

[KEDB hook fires, <10ms]
> Known Error KE-0005 | high | workaround available
>   Nested virtualization fails silently on Proxmox 8.x with AMD
>   Workaround: set cpu type to 'host' (not 'kvm64' or 'x86-64-v2-AES')
>   DO NOT try: enable_nested_virt sysfs toggle (reverts on reboot)
>   DO NOT try: modprobe kvm_amd nested=1 (requires host reboot, not VM)
>   Recurrence: 3 | Last seen: 2026-02-15

Claude: "Set cpu type to 'host'. Don't try the sysfs toggle — it doesn't persist..."
```

## 14. Referencias

### Investigacion KEDB/ITSM
- [ServiceNow Known Error Database implementation](https://www.servicenow.com/community/in-other-news/a-servicenow-implementation-of-the-known-error-database/ba-p/2291941)
- [KCS - Consortium for Service Innovation](https://www.serviceinnovation.org/kcs/)
- [KCS Solve Loop (v6 Practices Guide)](https://library.serviceinnovation.org/KCS/KCS_v6/KCS_v6_Practices_Guide/030/030)
- [ITIL Problem Management](https://wiki.en.it-processmaps.com/index.php/Problem_Management)
- [Google SRE: Postmortem Culture](https://sre.google/sre-book/postmortem-culture/)
- [DORA 2024 State of DevOps Report](https://dora.dev/research/2024/dora-report/)

### Herramientas SRE modernas
- [incident.io vs FireHydrant vs PagerDuty postmortems 2025](https://incident.io/blog/incident-io-vs-firehydrant-vs-pagerduty-automated-postmortems-2025)
- [Shoreline.io — Reinvents Runbooks](https://www.shoreline.io/blog/shoreline-io-reinvents-runbooks-with-industrys-first-purpose-built-notebooks-for-on-call-operations)
- [Jeli/PagerDuty Howie Guide](https://howie-guide.pagerduty.com/)
- [Rootly Auto-Sync Postmortems](https://rootly.com/sre/auto-sync-rootly-postmortems-to-slack-confluence-in-seconds)
- [PagerDuty SRE Agent with Memory](https://www.pagerduty.com/blog/ai/we-built-an-sre-agent-with-memory-and-its-transforming-incident-response/)

### Error matching sin LLM
- [Sentry Issue Grouping — Deterministic Fingerprinting](https://develop.sentry.dev/backend/application-domains/grouping/)
- [Sentry AI Noise Reduction (40% improvement over deterministic)](https://blog.sentry.io/how-sentry-decreased-issue-noise-with-ai/)
- [Bugsnag Error Grouping](https://docs.bugsnag.com/product/error-grouping/)
- [PagerDuty Intelligent Alert Grouping](https://support.pagerduty.com/main/docs/intelligent-alert-grouping)
- [Drain Log Parser (ICWS 2017)](https://netman.aiops.org/~peidan/ANM2023/6.LogAnomalyDetection/phe_icws2017_drain.pdf)
- [Drain3 — Production streaming log parser](https://github.com/logpai/Drain3)
- [Elasticsearch More Like This (TF-IDF based)](https://rebeccabilbro.github.io/intro-doc-similarity-with-elasticsearch/)
- [GitHub Blackbird — Sparse trigram code search](https://github.blog/engineering/architecture-optimization/the-technology-behind-githubs-new-code-search/)
- [Zoekt — Trigram code search in Go (Sourcegraph)](https://github.com/sourcegraph/zoekt)
- [Character n-gram TF-IDF — 92.4% F1 (ACL 2020)](https://aclanthology.org/2020.aespen-1.6/)
- [BM25 vs Dense Retrieval comparison](https://www.systemoverflow.com/learn/search-ranking/ranking-algorithms/bm25-vs-dense-retrieval-when-to-use-each)

### Bridge entity y knowledge graphs
- [Zep/Graphiti temporal knowledge graph](https://arxiv.org/abs/2501.13956)

### Daemon vs CLI — evidencia
- [gopls memory issues](https://github.com/golang/go/issues/47855)
- [rust-analyzer memory usage](https://github.com/rust-lang/rust-analyzer/issues/20028)
- [Datadog Agent memory leak](https://github.com/DataDog/datadog-agent/issues/29319)
- [Sentry Relay self-hosted issues](https://github.com/getsentry/self-hosted/issues/3566)
- [GitHub Copilot 750-file index limit](https://github.com/orgs/community/discussions/152490)
- [Atuin daemon discussion](https://forum.atuin.sh/t/moving-atuin-to-a-daemon/78)

### Escenarios aspiracionales — datos
- [Stack Overflow Developer Survey 2024](https://survey.stackoverflow.co/2024/)
- [IDC: Fortune 500 knowledge management losses](https://www.idc.com/)
- [Cloudflare November 2025 outage](https://blog.cloudflare.com/)
- [Cloudflare December 2025 outage](https://blog.cloudflare.com/)
- [DX Newsletter — Developer onboarding time](https://newsletter.getdx.com/)
- [Zoomin — Technical documentation freshness study](https://www.zoominsoftware.com/)
- [Wiz/EY — Log4Shell enterprise cloud impact (93%)](https://www.wiz.io/)
- [Claude Code memory prioritization (GitHub issue #8209)](https://github.com/anthropics/claude-code/issues/8209)

### Go libraries
- [modernc.org/sqlite — Pure Go SQLite with FTS5](https://pkg.go.dev/modernc.org/sqlite)
- [blevesearch/bleve — Full-text search + BM25 in Go](https://github.com/blevesearch/bleve)
- [sourcegraph/zoekt — Trigram code search in Go](https://github.com/sourcegraph/zoekt)

### LLM embebida (futuro v2)
- [hybridgroup/yzma — llama.cpp Go bindings via purego](https://github.com/hybridgroup/yzma)
- [onnxruntime-purego — ONNX inference sin CGO](https://pkg.go.dev/github.com/shota3506/onnxruntime-purego/onnxruntime)
- [Ollama Go API](https://pkg.go.dev/github.com/ollama/ollama/api)
- [all-MiniLM-L6-v2 (ONNX)](https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2)
- [Qwen2.5-3B-Instruct-GGUF](https://huggingface.co/Qwen/Qwen2.5-3B-Instruct-GGUF)

### Herramientas complementarias
- [Backscroll — Session & Plan Search CLI](../backscroll-session-search-cli.md) (research doc, mismo autor)
- [Rootline — File-based database engine](https://github.com/pablontiv/rootline)
