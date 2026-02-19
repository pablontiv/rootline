---
name: roadmap
description: |
  Framework de planificación AI-native: crear epics, stories o tasks.
  Acepta texto libre para descomposición autónoma de proyectos completos,
  o subcomandos explícitos (epic, story, task, pending, view).
  Tasks son unidades auto-contenidas con especificaciones técnicas
  y criterios de aceptación binarios.
  Usar cuando Pones pida "crear epic", "crear story", "crear task",
  "descomponer en features", "crear roadmap de X", "estructura de X",
  "planificar implementación de X", "qué sigue", "ver roadmap",
  "ver progreso", "qué falta", "Tasks pendientes",
  o texto libre describiendo trabajo a descomponer.
  Argumento: <texto libre> | epic|story|task|pending|view [path] [name] [description]
argument-hint: "<texto libre> | [epic|story|task|pending|view] [args]"
allowed-tools: Write, Read, Grep, Glob, Bash
disable-model-invocation: true
---

# /roadmap — Framework de Planificación AI-Native

## Modo de Operación

Este skill es **plan-mode aware**. Cuando `defaultMode: "plan"` está activo:

### Fase 1: Planificación (automática en plan mode)
1. Parsear `$ARGUMENTS` para determinar subcomando
2. Leer el guide file correspondiente
3. Ejecutar discovery y generar contenido completo en el plan file
4. Llamar `ExitPlanMode` para aprobación

### Fase 2: Ejecución (post-aprobación)
1. Crear archivos/directorios según el plan aprobado
2. Actualizar tabla en el README padre (cascading link)
3. Confirmar creación exitosa

---

## Modo Autónomo (default — sin subcomando explícito)

Cuando `$ARGUMENTS` NO empieza con `epic|story|task|pending|view`, activar modo de evaluación autónoma.

### Paso 1: Análisis de Intención

Leer `$ARGUMENTS` y determinar:
- **Qué proyecto/componente** se menciona
- **Qué profundidad** se pide (solo epics? hasta tasks?)
- **Qué documentación existe** del proyecto (README, intent docs, research, código)

### Paso 2: Absorber Contexto del Proyecto

Leer TODA la documentación disponible del proyecto mencionado:
- READMEs, intent docs, research docs
- Código existente (para dimensionar scope real)
- Dependencias y relaciones

Esto es fundamental — sin entender el proyecto completo, la descomposición será artificial.

### Paso 3: Aplicar Framework Autónomamente

**CRÍTICO**: El agente DEBE tomar decisiones usando los criterios del framework. NO preguntar al usuario cosas que el framework ya define.

Leer [framework-reference.md](framework-reference.md) y aplicar estos criterios de decisión:

| Nivel | Pregunta de corte | Criterio |
|-------|-------------------|----------|
| Epic | ¿Cuántos objetivos sistémicos distintos tiene? | Múltiples dominios → múltiples Epics |
| Feature | ¿Qué bloques pueden cerrarse independientemente? | Milestone técnico real (anti-inflación: 3-5 Features, no 10) |
| Story | ¿Qué capacidades nuevas existen? | Antes/después claro, testeable, no ejecutable en 1 sesión |
| Task | ¿Qué puede hacer un agente en 1 sesión? | 6 condiciones de task-guide.md |

**Criterios de escala** (decision tree numérico):

```
1. ¿Cuántos OBJETIVOS SISTÉMICOS distintos tiene el proyecto?
   → 1 objetivo claro → 1 Epic
   → 2-3 objetivos independientes → 2-3 Epics
   → Señal de separación: si dos Features NO comparten dependencias
     ni contribuyen al mismo objetivo → son Epics distintos

2. ¿Cuántos MILESTONES independientes tiene cada Epic?
   → Target: 3-5 Features por Epic
   → > 7 Features → el Epic es demasiado grande, dividir
   → Feature con 1 sola Story → absorber en Feature vecino

3. ¿Cuántas CAPACIDADES NUEVAS tiene cada Feature?
   → Target: 1-4 Stories por Feature
   → Story con > 5 Tasks → probablemente es un Feature, elevar nivel

4. ¿Cuántas SESIONES requiere cada Story?
   → Target: 1-5 Tasks por Story
   → Task que requiere > 1 sesión → dividir en Tasks más pequeños
```

**Señales de que un nivel está mal**:
- Nivel con 1 solo hijo → probablemente no merece ser un nivel (absorber)
- Nivel con > 10 hijos → demasiado ancho, necesita un nivel intermedio
- Todos los Tasks del mismo tipo → posible Feature artificial (¿realmente es un milestone?)

### Paso 4: Generar Descomposición en Plan File

Presentar la estructura completa propuesta con árbol jerárquico:

```
E01: [Objetivo sistémico 1]
├── F01: [Milestone]
│   ├── S001: [Capacidad]
│   │   ├── T001: [tarea atómica] (tipo: X)
│   │   └── T002: [tarea atómica] (tipo: X)
│   └── S002: [Capacidad]
│       └── T001: [tarea atómica] (tipo: X)
└── F02: [Milestone]
    └── S001: [Capacidad]
        └── T001: [tarea atómica] (tipo: X)

E02: [Objetivo sistémico 2]
└── ...
```

Para cada Task incluir: nombre, tipo, descripción de 1 línea.

### Paso 4.5: Validación de Completitud

**OBLIGATORIO** antes de presentar. Verificar:

1. **Traceability ascendente**: Cada Task → contribuye a su Story "Después"
   → cada Story → contribuye a su Feature Objetivo
   → cada Feature → avanza la Intención del Epic.
   Si un Task no traza a ningún objetivo superior → eliminar o reubicar.

2. **Completeness descendente**: ¿Completar TODOS los Features logra la Intención del Epic?
   Si hay aspectos de la Intención NO cubiertos → crear Feature faltante.

3. **No-overlap**: ¿Dos Features o Stories cubren lo mismo? → fusionar.

4. **Dependency chain**: ¿Features tienen dependencias entre sí?
   → Documentar orden de ejecución en el plan.

5. **Sanity check numérico**: Verificar contra criterios de escala (Paso 3).

### Paso 5: Presentar para Aprobación (NO para definición)

El plan se presenta como **propuesta fundamentada**, no como pregunta abierta.
- El agente YA tomó las decisiones de granularidad
- El usuario aprueba, ajusta, o rechaza — pero no define desde cero
- Si hay ambigüedad REAL (no resuelta por el framework), ENTONCES preguntar

### Anti-patrones

- ❌ "¿Debería haber 1 Epic o varios?" — El framework ya define cuándo
- ❌ "¿Qué opina de esta estructura?" — Presentar la estructura, no pedir que la diseñe
- ❌ Proponer 1 Epic para un producto completo — Escala mal
- ❌ Preguntar por cada nivel — Generar TODO y presentar junto

### Paso 6: Crear Estructura (post-aprobación)

Una vez aprobada, crear TODOS los archivos usando los templates de:
- [epic-guide.md](epic-guide.md) para READMEs de Epic y Feature
- [story-guide.md](story-guide.md) para READMEs de Story
- [task-guide.md](task-guide.md) para archivos de Task

---

## Subcomandos

### `/roadmap epic <name> <description>`

Descomponer intención estratégica en features y stories.

**Leer**: [epic-guide.md](epic-guide.md) para workflow completo.

**Output**: `docs/epics/EXX-name/README.md` + subdirectorios Feature con READMEs skeleton.

**Ejemplo**: `/roadmap epic disaster-recovery Garantizar recuperación completa en < 2 horas`

---

### `/roadmap story <feature-path> <name> <description>`

Crear story como contrato semántico con estructura antes/después.

**Leer**: [story-guide.md](story-guide.md) para workflow completo.

**Output**: `docs/epics/.../SXXX-name/README.md` bajo el Feature indicado.

**Ejemplo**: `/roadmap story E01/F13 layer3-validation k8s workloads validados en CST`

---

### `/roadmap task <story-path> <name> <description>`

Crear task AI-ready con criterios de aceptación binarios.

Cada Task tiene un campo `Tipo` que determina su naturaleza (servicio-docker, modulo-sistema, operacion-sistema, lxc, vm, modulo-infraestructura, host-script, instance-script, documentation). Tasks con tipos IaC incluyen una sección `Especificacion Tecnica` con YAML type-specific, haciendo cada Task auto-contenida (spec + ejecución en un solo archivo).

**Leer**: [task-guide.md](task-guide.md) para workflow completo.

**Output**: `docs/epics/.../TXXX-name.md` bajo la Story indicada.

**Ejemplo**: `/roadmap task E01/F13/S001 add-k8s-phase Agregar fase k8s al playbook CST`

---

### `/roadmap pending`

Mostrar solo Tasks pendientes en formato tabla, agrupados por Epic/Feature.

**Procedimiento**: Ejecuta este script Python:

```python
import os, re, glob, subprocess

BASE = "/opt/homeserver/automation"

def parse_frontmatter(content):
    """Extract YAML frontmatter fields from Task file."""
    fm = {}
    if content.startswith('---'):
        end = content.find('---', 3)
        if end > 0:
            for line in content[3:end].strip().splitlines():
                if ':' in line:
                    key, val = line.split(':', 1)
                    fm[key.strip()] = val.strip()
    return fm

def extract_story(filepath):
    """Derive Story context from filesystem path."""
    parts = filepath.split('/')
    for p in parts:
        if re.match(r'S\d{3}-', p):
            return p
    return "?"

def extract_epic_feature(filepath):
    """Derive Epic/Feature from filesystem path."""
    parts = filepath.split('/')
    epic = feature = ""
    for p in parts:
        if re.match(r'E\d{2}-', p): epic = p.split('-', 1)[0]
        if re.match(r'F\d{2}-', p): feature = p.split('-', 1)[0]
    return f"{epic}/{feature}" if epic else "?"

tasks = []
for f in glob.glob(f"{BASE}/docs/epics/**/T[0-9][0-9][0-9]-*.md", recursive=True):
    with open(f) as h: content = h.read(3000)
    fm = parse_frontmatter(content)
    estado = fm.get('estado', '')
    # Match Pending (including "Pending (blocked by ...)" variants) and Especificado
    if not re.match(r'(Pending|Especificado)', estado):
        continue
    tipo = fm.get('tipo', '-')
    # Extract Task ID and title from heading
    title_match = re.search(r'^# T\d{3}:\s*(.+)$', content, re.M)
    title = title_match.group(1)[:40] if title_match else "Sin titulo"
    # Extract TXXX from filename
    task_id = re.search(r'(T\d{3})', os.path.basename(f)).group(1)
    story = extract_story(f)
    location = extract_epic_feature(f)
    tasks.append((location, story, task_id, title, tipo, estado, f))

# Detect new (untracked) Task files via git status
new_files = set()
try:
    r = subprocess.run(['git', 'status', '--porcelain'], capture_output=True, text=True, cwd=BASE)
    for l in r.stdout.splitlines():
        if '??' in l and '/T' in l and l.strip().endswith('.md'):
            m = re.search(r'(T\d{3})', l)
            if m: new_files.add(m.group(1))
except Exception: pass

# Group by Epic/Feature
from collections import defaultdict
grouped = defaultdict(list)
for loc, story, tid, title, tipo, estado, fpath in tasks:
    grouped[loc].append((story, tid, title, tipo, estado, tid in new_files))

print(f"Tasks Pendientes ({len(tasks)})\n")
for loc in sorted(grouped.keys()):
    items = sorted(grouped[loc])
    print(f"### {loc}\n")
    print("| Story | Task | Titulo                                   | Tipo              | Estado    |")
    print("|-------|------|------------------------------------------|-------------------|-----------|")
    for story, tid, title, tipo, estado, is_new in items:
        story_short = re.sub(r'S(\d{3})-.*', r'S\1', story)
        marker = " *new*" if is_new else ""
        print(f"| {story_short:<5} | {tid}  | {title:<40} | {tipo:<17} | {estado:<9}{marker} |")
    print()
```

Presenta el output tal cual, sin modificaciones.

---

### `/roadmap` o `/roadmap view`

Mostrar árbol jerárquico del estado actual (read-only).

**Procedimiento**:
1. Escanear `docs/epics/` recursivamente con Glob
2. **Tasks**: Leer YAML frontmatter de cada `T*.md` para extraer `estado:`
3. **Stories**: Contar tasks completados/total → `[n/m]`
4. **Features**: Sumar tasks completados/total de todas sus Stories → `[n/m]`
5. **Epics**: Sumar tasks completados/total de todas sus Features → `[n/m]`
6. Renderizar árbol ASCII con `[n/m]`

**Reglas de renderizado**:
- `✅` para nodos con `n/n` donde n > 0 (todas las tasks completadas)
- `○` para nodos con `[0/0]` (sin tasks, pendiente de descomponer)
- Tasks → siguen mostrando `[Pending]` / `[Completado]` como hojas del árbol
- Siempre expandir todas las tasks, incluso en Stories completadas

**Formato de salida**:
```
docs/epics/
├── E01: Infrastructure Foundation [0/4]
│   ├── ○ F01: Host Bootstrap [0/0]
│   ├── ○ F02: Proxmox Provisioning [0/0]
│   ├── ○ F03: Bootstrap Orchestrator [0/0]
│   ├── F13: CST Testing [0/2]
│   │   └── S001: CST Layer 3 Validation [0/2]
│   │       ├── T001: add-k8s-workload-phase [Pending]
│   │       └── T002: validate-cst-layer3-pass [Pending]
│   └── F14: DR Runbooks [0/2]
│       └── S001: DR-001 Production Execution [0/2]
│
├── E02: AI-Native Development Framework [8/14]
│   ├── ✅ F01: Task Template Extension [3/3]
│   │   └── ✅ S001: IaC Task Types [3/3]
│   │       ├── T001: extend-task-guide [Completado]
│   │       ├── T002: update-framework-reference [Completado]
│   │       └── T003: update-roadmap-skill [Completado]
│   ├── F02: Skill/Hook Migration [5/8]
│   │   ├── ✅ S001: Implementation Skills [5/5]
│   │   │   ├── T001: migrate-service-skill [Completado]
│   │   │   ├── T002: migrate-module-skill [Completado]
│   │   │   ├── T003: migrate-operation-skill [Completado]
│   │   │   ├── T004: migrate-instance-skill [Completado]
│   │   │   └── T005: migrate-script-skills [Completado]
│   │   └── S002: Query Skills & Hooks [0/3]
│   ├── F03: Rules & Docs Cleanup [0/6]
│   │   ├── S001: Rules & CLAUDE.md [0/4]
│   │   └── S002: Dead Code Removal [0/2]
│   └── ○ F04: PRD Data Migration [0/0]
```

---

## Lógica Común

### Auto-numbering

Para cada nivel, detectar el próximo número disponible:

```bash
# Epics: próximo EXX
ls -d docs/epics/E[0-9][0-9]-* 2>/dev/null | sort -V | tail -1

# Features: próximo FXX dentro del Epic
ls -d docs/epics/EXX-name/F[0-9][0-9]-* 2>/dev/null | sort -V | tail -1

# Stories: próximo SXXX dentro del Feature
ls -d docs/epics/.../SXXX-* 2>/dev/null | sort -V | tail -1

# Tasks: próximo TXXX dentro de la Story
ls docs/epics/.../T[0-9][0-9][0-9]-*.md 2>/dev/null | sort -V | tail -1
```

### Verificación de Padre

SIEMPRE verificar que el directorio padre existe antes de crear:
- `/roadmap story E01/F13 ...` → verificar `docs/epics/E01-*/F13-*/README.md` existe
- `/roadmap task E01/F13/S001 ...` → verificar `docs/epics/E01-*/F13-*/S001-*/README.md` existe

Si no existe → informar a Pones y sugerir crearlo primero.

### Cascading Links

Después de crear un artefacto, actualizar la tabla en el README padre:
- Task creado → agregar fila en la tabla "Tasks" del Story README (solo Task + Descripcion, sin Estado)
- Story creada → agregar fila en la sección "Stories" del Feature README (sin Estado)

**Nota**: Las tablas NO incluyen columna Estado. El estado se lee del YAML frontmatter de cada Task y se deriva para Stories/Features en `/roadmap view`.

---

## Referencia

- Ver [framework-reference.md](framework-reference.md) para el documento completo del marco de trabajo
- Ver `.claude/rules/planning-framework.md` para principios siempre activos
- Templates canónicos: `docs/epics/E01-infrastructure-foundation/`
