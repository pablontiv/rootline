---
name: roadmap
description: |
  AI-native planning framework for autonomous project decomposition.
  Accepts free text to decompose into epics, features, stories, and tasks.
  Subcommands: pending, loop, plan.
  Tasks are self-contained units with technical specs and binary acceptance criteria.
  This skill should be used when the user says "descomponer en features",
  "crear roadmap de X", "estructura de X",
  "planificar implementación de X", "qué sigue", "ver roadmap",
  "ver progreso", "qué falta", "tasks pendientes",
  "loop de tasks", "ejecutar pendientes", "implementar tasks",
  "roadmap loop", "ejecutar roadmap",
  "crear roadmap del plan", "materializar plan",
  or provides free text describing work to decompose.
argument-hint: "<texto libre> | [pending|loop|plan] [args]"
allowed-tools:
  - Write
  - Read
  - Grep
  - Glob
  - Bash
  - TaskCreate
  - TaskList
  - TaskUpdate
  - TaskGet
  - Skill
  - AskUserQuestion
  - ExitPlanMode
---

# /roadmap — Framework de Planificación AI-Native

## Modo de Operación

Este skill es **plan-mode aware**. Cuando `defaultMode: "plan"` está activo:

### Fase 1: Planificación (automática en plan mode)
1. Parsear `$ARGUMENTS` para determinar subcomando
2. Leer el guide file correspondiente
3. Ejecutar discovery y generar contenido completo en el plan file
4. Llamar `ExitPlanMode` para aprobación

### Fase 2: Post-aprobación

Después de que el usuario aprueba el plan, informarle que puede ejecutar `/roadmap plan` para crear los archivos del roadmap.

---

## Modo Autónomo (default — sin subcomando explícito)

Cuando `$ARGUMENTS` NO empieza con `pending|loop|plan`, activar modo de evaluación autónoma.

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

Apply the **scale criteria and decision tree** from [framework-reference.md](framework-reference.md) — targets: 3-5 Features/Epic, 1-4 Stories/Feature, 1-5 Tasks/Story. Split when exceeding limits, absorb when only 1 child exists.

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

### Paso 6: Informar siguiente paso

Después de la aprobación, informar al usuario que puede ejecutar `/roadmap plan` para materializar la estructura como archivos .md.

---

## Subcomandos

### `/roadmap pending`

Vista jerárquica filtrada: solo Features con trabajo pendiente.

**Procedimiento**:
1. Ejecutar `rootline tree docs/epics/ --where "tipo not in ['feature', 'historia']" --where "estado != 'Completed'" --output table`
2. Ejecutar `rootline stats docs/epics/ --where "tipo not in ['feature', 'historia']" --where "estado != 'Completed'" --output table`

Presenta ambos outputs al usuario.

---

### `/roadmap plan`

Materializar el plan aprobado más reciente como archivos de roadmap.

**Cuándo usar**: Después de aprobar un plan en plan mode que contiene una descomposición de roadmap (estructura epic/feature/story/task).

**Procedimiento**:
1. Buscar el plan file más reciente en `~/.claude/plans/` (ordenar por fecha de modificación, tomar el más reciente)
2. Leer el contenido del plan file
3. Parsear la estructura jerárquica del plan (epics, features, stories, tasks)
4. Para cada artefacto, crear archivos .md usando los templates de:
   - [epic-guide.md](epic-guide.md) para READMEs de Epic y Feature
   - [story-guide.md](story-guide.md) para READMEs de Story
   - [task-guide.md](task-guide.md) para archivos de Task
5. Después de cada Write, ejecutar `rootline validate <path>`
6. Si falla, `rootline fix <path>` como fallback
7. Actualizar tablas en READMEs padre (cascading links)
8. Confirmar creación exitosa

**IMPORTANTE**: Solo crear archivos .md de planificación. NO implementar el trabajo descrito en los tasks — eso lo hace `/roadmap loop`.

---

### `/roadmap` (sin argumentos)

Mostrar árbol jerárquico y resumen estadístico del estado actual (read-only).

**Procedimiento**:
1. Ejecutar `rootline tree docs/epics/ --where "tipo not in ['feature', 'historia']" --output table`
2. Ejecutar `rootline stats docs/epics/ --where "tipo not in ['feature', 'historia']" --output table`

Presenta ambos outputs al usuario sin modificaciones.

---

### `/roadmap loop [--filter PATTERN] [--max N]`

Ejecutar Tasks pendientes en loop con confirmación entre cada uno.

**Opciones**:
- `--filter PATTERN`: Filtrar por path (ej: `E02/F04`, `E01`)
- `--max N`: Limitar a N tasks

**Procedimiento**:

#### Fase 1: Discovery

1. Ejecutar `rootline graph --check docs/epics/` para validar dependencias antes de empezar
   - Si hay ciclos → reportar y **parar** (dependencias circulares impiden ejecución)
   - Si hay broken links → reportar como warning (pueden ser tasks aún no creados)
2. Ejecutar `rootline query docs/epics/ --where "tipo not in ['feature', 'historia']" --where "estado in ['Specified', 'In Progress']" --output table` para obtener tasks pendientes
3. Si `--filter PATTERN` proporcionado, filtrar resultados por Epic/Feature path match
4. Si `--max N`, tomar solo los primeros N tasks
5. Mostrar tabla de tasks encontradas al usuario

#### Fase 2: TodoList Setup

Para cada task encontrada, crear entrada con `TaskCreate`:
- **subject**: `TXXX: título`
- **description**: `Path: <filepath> | Tipo: <tipo>`
- **activeForm**: `Implementando TXXX`

Mostrar TodoList con `TaskList`.

#### Fase 3: Loop de Ejecución

Para cada task en orden:

1. **Verificar dependencias**: Leer el archivo .md del task y buscar `[[blocks:TXXX-name]]` en el body.
   Para cada dependencia encontrada:
   - Buscar el task referenciado y verificar que su frontmatter tiene `estado: Completed`
   - Si alguna dependencia no está Completed → **skip** con mensaje: `⏭️ Bloqueado por: TXXX (estado: Pending)`
   - Tasks bloqueados se reintentarán al final de la cola

2. **Marcar inicio**: `TaskUpdate` → status: `in_progress`

3. **Leer Task**: `Read` del archivo .md completo para entender qué pide

4. **Implementar**:
   - Si el Task tiene `tipo:` en frontmatter que corresponde a un skill
     conocido del proyecto, invocarlo via `Skill` tool (ej: tipo `servicio-docker` → `/service TXXX`)
   - Si no tiene skill asociado, implementar directamente siguiendo
     las instrucciones del Task

5. **Commit+Push** (centralizado, NO delegado a skills hijos):
   - Identificar archivos modificados/creados por la implementación
   - `git add` archivos relevantes (específicos, no `git add .`)
   - `git commit` con mensaje en formato **conventional commits**: `type(scope): description`
     - Elegir `type` según el contenido del task: `feat` (nueva funcionalidad), `fix` (corrección), `test` (tests), `docs` (documentación), `refactor` (reestructuración), `ci` (CI/CD), `chore` (mantenimiento), `perf` (rendimiento), `style` (formato)
     - El hook `.githooks/commit-msg` rechazará mensajes que no sigan el formato
   - `git push`

6. **Verificar ACs**:
   - Leer sección "Criterios de Aceptación" del Task .md
   - Ejecutar CADA verificación documentada (comandos, checks, observables)
   - Reportar resultado por AC: ✅ PASS / ❌ FAIL
   - Si algún AC falla → reportar y **parar** (bug encontrado)

7. **Marcar completado**: `TaskUpdate` → status: `completed`

8. **Resumen de iteración**:
   ```
   📊 ITERACIÓN N/TOTAL
   ├─ Task: TXXX - título
   ├─ Resultado: ✅/❌
   ├─ ACs: N/M passed
   ├─ Commit: hash
   └─ Siguiente: TXXX+1 - título
   ```

9. **Confirmar**: `AskUserQuestion` con opciones:
   - Sí, continuar (Recommended)
   - Saltar siguiente y continuar
   - Parar aquí

10. **Reintentar bloqueados**: Al terminar la cola, si quedan tasks que fueron skipped por dependencias bloqueadas y ahora sus dependencias están Completadas → reintentar. Si ningún task progresó en la pasada → parar (deadlock de dependencias).

#### Fase 4: Resumen Final

Al terminar todas las tasks o al parar:

```
📊 RESUMEN LOOP
├─ Tasks completadas: N/TOTAL
├─ Tasks saltadas: M
├─ ACs: total passed / total
├─ Commits: lista de hashes
└─ Tasks restantes: lista (si las hay)
```

---

## Lógica Común

### Auto-numbering

Para cada nivel, usar `rootline describe` con el campo `schema.id.next`:

```bash
# Requiere .stem con id: {type: sequence, prefix: X, digits: N} en cada nivel

# Epics: próximo EXX
rootline describe docs/epics/ --field schema.id.next

# Features: próximo FXX dentro del Epic
rootline describe docs/epics/EXX-name/ --field schema.id.next

# Stories: próximo SXXX dentro del Feature
rootline describe docs/epics/.../FXX-name/ --field schema.id.next

# Tasks: próximo TXXX dentro de la Story
rootline describe docs/epics/.../SXXX-name/ --field schema.id.next
```

El comando retorna el próximo identificador directamente (ej: `"T004"`).

### Verificación de Padre

SIEMPRE verificar que el directorio padre existe antes de crear un artefacto:
- Verificar con `rootline describe docs/epics/<path>/` que el directorio destino existe

Si no existe → informar al usuario y sugerir crearlo primero.

### Cascading Links

Después de crear un artefacto, actualizar la tabla en el README padre:
- Task creado → agregar fila en la tabla "Tasks" del Story README (solo Task + Descripcion, sin Estado)
- Story creada → agregar fila en la sección "Stories" del Feature README (sin Estado)

**Nota**: Las tablas NO incluyen columna Estado. El estado se lee del YAML frontmatter de cada Task y se deriva para Stories/Features en `/roadmap`.

---

## Comandos Rootline de Referencia

| Comando | Cuándo usarlo en el skill |
|---------|--------------------------|
| `rootline validate <path>` | Después de crear/editar archivos .md — verificar contra .stem |
| `rootline fix <path>` | Cuando validate falla — corregir automáticamente |
| `rootline describe <dir> --field schema.id.next` | Auto-numbering: obtener próximo ID en cualquier nivel |
| `rootline new <path>` | Scaffolding: crear archivo con frontmatter correcto según .stem |
| `rootline query <path> --where "expr"` | Discovery: buscar records por frontmatter (estado, tipo, etc.) |
| `rootline tree <path> --where "expr" --output table` | Vista jerárquica filtrada: `/roadmap pending` |
| `rootline stats <path> --where "expr" --output table` | Resumen estadístico filtrado por expresión |
| `rootline graph <path> --where "expr" --check` | Grafo de dependencias filtrado |

## Referencia

- Ver [framework-reference.md](framework-reference.md) para el documento completo del marco de trabajo
- Templates canónicos: `docs/epics/E01-infrastructure-foundation/`
