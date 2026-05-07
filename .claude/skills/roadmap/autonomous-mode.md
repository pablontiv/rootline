# Modo Autonomo — Descomposicion de proyecto

Cuando `$ARGUMENTS` NO empieza con `pending|loop|plan`, activar modo de evaluacion autonoma.

## Paso 0: Bootstrap (obligatorio — ejecutar PRIMERO)

Ejecutar la seccion "Configuracion del Proyecto — Bootstrap Obligatorio" de SKILL.md. Leer o crear `.claude/roadmap.local.md` ANTES de cualquier analisis. Este paso NO requiere rootline.

## Paso 1: Analisis de Intencion

**Workspace mode**: Resolver repo target desde `$ARGUMENTS`.
Si menciona un nombre que matchea un repo en `<repos>` (ej: "planificar backscroll E16") → usar ese repo y su config.
Si no matchea ningún repo → `AskUserQuestion` para seleccionar repo.
Una vez resuelto, usar `<abs-roadmap-root>` y `<repo-path>` de ese repo para todo el análisis.

Leer `$ARGUMENTS` y determinar:
- **Que proyecto/componente** se menciona
- **Que profundidad** se pide (solo epics? hasta tasks?)
- **Que documentacion existe** del proyecto (README, intent docs, research, codigo)

## Paso 2: Absorber Contexto del Proyecto

Leer TODA la documentacion disponible del proyecto mencionado:
- READMEs, intent docs, research docs
- Codigo existente (para dimensionar scope real)
- Dependencias y relaciones

Esto es fundamental — sin entender el proyecto completo, la descomposicion sera artificial.

## Paso 2.5: Filtro de Vocabulario (obligatorio)

El roadmap es un artefacto de **implementacion**. Si el contexto proviene de `/discover`, `/hypothesize`, o cualquier documento de investigacion, traducir TODO el vocabulario antes de descomponer:

| Vocabulario de investigacion (NUNCA usar) | Vocabulario de implementacion (usar) |
|-------------------------------------------|--------------------------------------|
| Linea de investigacion, hipotesis, premisa | Objetivo, capacidad, requisito |
| Categoria CAP-XX, Mecanismo M-XX, CD-XX | Nombre descriptivo del dominio tecnico |
| Fase, ciclo PAOR, observacion, reflexion | (eliminar — no aplica) |
| Premisa empirica, evidencia, falsacion | Requisito tecnico, constraint |
| Veredicto Go/No-Go, senal confirmatoria | (eliminar — la decision ya fue tomada) |
| Codigo interno LI-XX, H-XX | (eliminar — usar nombre descriptivo) |

**Test de aislamiento**: Un desarrollador que NO participo en la investigacion entiende cada nombre de Epic, Feature, Story y Task sin consultar otro documento? Si no → reescribir.

**Anti-patrones**:
- Epic "Implementar resultados de LI-03" → Epic "Sistema de validacion de documentos"
- Story "Validar premisa CAP-07 sobre M6" → Story "Parser de frontmatter YAML"
- Task "Falsear hipotesis de rendimiento" → Task "Benchmark de parsing con 1000 archivos"

## Paso 2.6: Formalizar Contratos

**ANTES de descomponer**, para cada Epic identificado, definir:

1. **Postcondiciones** (2-3 constraints observables): Condiciones que seran verdad cuando el Epic se complete. Cada postcondicion DEBE incluir un **comando o procedimiento de verificacion** concreto. "Produce un PDF valido" no es observable — "`md2pdf input.md -o out.pdf && pdfinfo out.pdf | grep -q Pages`" si lo es.
2. **Invariantes**: Reglas que ningun Feature/Story/Task puede violar durante su ejecucion. Cada invariante incluye su **procedimiento de verificacion**. Ejemplo: *"Los workflows existentes siguen funcionando sin regresion"* → verificar: `ansible-playbook site.yml --check` retorna 0.
3. **Out of scope**: Limites explicitos que previenen scope creep.

**Formato en plan file — Constraint Map:**

```markdown
## Constraint Map

| Postcondicion | Features que la satisfacen | Descripcion |
|---------------|---------------------------|-------------|
| P1: ...       | F01, F03                  | ...         |
| P2: ...       | F02                       | ...         |

## Invariantes

- INV1: ...
- INV2: ...
```

**Validacion bidireccional** (obligatoria):
- Toda postcondicion tiene al menos un Feature que la satisface
- Todo Feature satisface al menos una postcondicion
- Si algun Feature no satisface ninguna postcondicion → eliminar o reubicar
- Si alguna postcondicion no tiene Feature → crear Feature faltante

## Paso 3: Aplicar Framework Autonomamente

**CRITICO**: El agente DEBE tomar decisiones usando los criterios del framework. NO preguntar al usuario cosas que el framework ya define.

Leer [framework-reference.md](framework-reference.md) y aplicar estos criterios de decision:

| Nivel | Pregunta de corte | Criterio |
|-------|-------------------|----------|
| Epic | Cuantos objetivos sistemicos distintos tiene? | Multiples dominios → multiples Epics |
| Feature | Que bloques pueden cerrarse independientemente? Satisface >= 1 postcondicion del Epic | Milestone tecnico real (anti-inflacion: 3-5 Features, no 10) |
| Story | Que capacidades nuevas existen? | Antes/despues claro, testeable, no ejecutable en 1 sesion |
| Task | Que puede hacer un agente en 1 sesion? | 7 condiciones de task-guide.md |

### Criterios de Escala (obligatorios — verificar ANTES de presentar)

| Nivel | Target | Minimo | Maximo | Si excede → | Si solo 1 hijo → |
|-------|--------|--------|--------|-------------|-------------------|
| Features/Epic | 3-5 | 2 | 7 | Dividir Epic | Absorber en Epic vecino |
| Stories/Feature | 1-4 | 1 | 4 | Dividir Feature | Aceptable (Feature simple) |
| Tasks/Story | 1-5 | 1 | 5 | Dividir Story | Aceptable (Story atomica) |

**Sustancia minima por Epic**: Si un Epic tiene < 3 Features, cada Feature DEBE tener >= 2 Stories. Un Epic con 2 Features de 1 Story cada uno (≈ 4 Tasks) no tiene sustancia suficiente — absorber como Feature en un Epic vecino o enriquecer con mas Stories. Minimo viable: 6 Tasks por Epic.

**Heuristicas de granularidad:**

- **Feature con 1 Story** → Probablemente no es Feature, absorber en Feature vecino
- **Story que se ejecuta en 1 sesion** → Probablemente es un Task, no una Story
- **Task con mas de 5 archivos a modificar** → Demasiado grande, dividir
- **Epic con nombre generico** (Advanced, Misc, DX, Improvements) → No tiene intencion clara, replantear
- **Epic con < 6 Tasks totales** → No tiene sustancia, absorber en Epic vecino o enriquecer Features

**Ejemplo concreto de buena vs mala granularidad:**

```
MAL: 1 Epic con 8 Features de 1 Story cada uno
   E01: Platform Improvements
   ├── F01: Config → S001: Add config (1 task)
   ├── F02: Logging → S001: Add logging (1 task)
   └── ... (6 mas igual)

BIEN: 2 Epics enfocados con Features sustanciales
   E01: Configuration System
   ├── F01: Schema Validation (S001, S002 → 4 tasks)
   └── F02: Multi-env Support (S001, S002 → 3 tasks)
   E02: Observability
   ├── F01: Structured Logging (S001 → 3 tasks)
   └── F02: Health Monitoring (S001, S002 → 4 tasks)
```

## Paso 4: Generar Descomposicion en Plan File

Presentar la estructura completa propuesta con arbol jerarquico:

```
E01: [Objetivo sistemico 1]
├── F01: [Milestone]
│   ├── S001: [Capacidad]
│   │   ├── T001: [tarea atomica] (tipo: X)
│   │   └── T002: [tarea atomica] (tipo: X)
│   └── S002: [Capacidad]
│       └── T001: [tarea atomica] (tipo: X)
└── F02: [Milestone]
    └── S001: [Capacidad]
        └── T001: [tarea atomica] (tipo: X)

E02: [Objetivo sistemico 2]
└── ...
```

Para cada Task incluir: nombre, tipo, descripcion de 1 linea.

**Constraint Map** (obligatorio en plan file):

```markdown
## Constraint Map

| Postcondicion Epic | Features | Descripcion |
|----|----------|-------------|
| P1: ... | F01, F03 | ... |
| P2: ... | F02 | ... |
```

## Paso 4.5: Validacion de Completitud

**OBLIGATORIO** antes de presentar. Verificar:

1. **Traceability ascendente**: Cada Task → contribuye a su Story "Despues"
   → cada Story → contribuye a su Feature Objetivo
   → cada Feature → avanza la Intencion del Epic.
   Si un Task no traza a ningun objetivo superior → eliminar o reubicar.

2. **Completeness por contratos**: Cada postcondicion del Epic tiene >= 1 Feature que la satisface. Cada milestone de Feature tiene >= 1 Story que lo cubre. Cada criterio de Story tiene >= 1 Task AC que lo implementa. Si algun nivel no tiene cobertura → crear artefacto faltante.

3. **No-overlap**: Dos Features o Stories cubren lo mismo? → fusionar.

4. **Dependency chain**: Features tienen dependencias entre si?
   → Documentar orden de ejecucion en el plan.

5. **Sanity check numerico**: Verificar contra criterios de escala (Paso 3).

6. **Invariant propagation check**: Invariantes del Epic aparecen en sus Features (heredados). Invariantes de Features fluyen a sus Stories. Tasks los preservan via seccion "Preserva". Si un invariante no se propaga → agregarlo al nivel faltante.

## Paso 5: Presentar para Aprobacion (NO para definicion)

El plan se presenta como **propuesta fundamentada**, no como pregunta abierta.
- El agente YA tomo las decisiones de granularidad
- El usuario aprueba, ajusta, o rechaza — pero no define desde cero
- Si hay ambiguedad REAL (no resuelta por el framework), ENTONCES preguntar

### Anti-patrones

- "Deberia haber 1 Epic o varios?" — El framework ya define cuando
- "Que opina de esta estructura?" — Presentar la estructura, no pedir que la disene
- Proponer 1 Epic para un producto completo — Escala mal
- Preguntar por cada nivel — Generar TODO y presentar junto

## Paso 6: Informar siguiente paso

Despues de la aprobacion, informar al usuario que puede ejecutar `/roadmap plan` para materializar la estructura como archivos .md.
