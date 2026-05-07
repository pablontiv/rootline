# Task Guide — Crear Task AI-Ready

## Workflow

### Paso 1: Parsear argumentos

Extraer:

- **task-name**: slug kebab-case, ej. `add-k8s-phase`.
- **descripción**: qué debe hacer el agente.
- **outcome-path opcional**: directorio `OXX-*` si la task pertenece a un Outcome.

### Paso 2: Determinar destino

- Si pertenece a un Outcome: `<roadmap-root>/OXX-name/TXXX-task-name.md`.
- Si es directa: `<roadmap-root>/TXXX-task-name.md`.

Verificar el directorio destino con:

```bash
rootline describe <directorio>/
```

### Paso 3: Auto-numbering

Si la task va dentro de un Outcome:

```bash
rootline describe <outcome-dir>/ --field schema.id.next
```

Si la task va directa en la raíz del roadmap:

```bash
find <roadmap-root>/ -maxdepth 1 -type f -name 'T[0-9][0-9][0-9]-*.md' -printf '%f\n' | sort
```

Tomar el mayor `TXXX` y sumar 1; si no hay ninguno, usar `T001`.

### Paso 4: Crear archivo

```bash
rootline new <directorio>/TXXX-task-name.md
```

Editar el contenido sin inventar campos fuera del `.stem` efectivo.

### Paso 5: Actualizar README padre si existe

Si la task pertenece a un Outcome, agregar fila en `OXX-*/README.md`:

```markdown
| [TXXX](TXXX-task-name.md) | Descripción breve |
```

## Dependencias

Usar `blocked_by` en la task bloqueada, siempre con path relativo explícito:

```markdown
[[blocked_by:./T001-prerequisite.md]]
```

Si la dependencia está en otro Outcome:

```markdown
[[blocked_by:../O01-setup/T001-prerequisite.md]]
```

Significa: la task actual no puede ejecutarse hasta que esa task esté completada.

No usar targets bare como `[[blocked_by:T001-prerequisite]]`: rootline solo puede resolverlos por basename único y se rompen si hay duplicados.

## Template: Task File

```markdown
---
estado: Specified
tipo: task
---
# TXXX: [Descripción accionable]

**Outcome**: [OXX Nombre](README.md) <!-- omitir si es task directa -->
**Contribuye a**: [criterio de éxito del Outcome o resultado directo esperado]

[[blocked_by:./TXXX-prerequisite.md]] <!-- omitir si no hay dependencia -->

## Preserva

- INV1: [invariante a mantener]
  - Verificar: [comando o procedimiento]

## Contexto

[Contexto suficiente para que un agente ejecute esta task leyendo solo este archivo.]

## Alcance

**In**:
1. [acción concreta]
2. [acción concreta]

**Out**:
- [límite explícito]

## Estado inicial esperado

- [precondición observable]

## Criterios de Aceptación

- [AC binario con comando/check esperado]
- [AC binario con comando/check esperado]

## Fuente de verdad

- [paths a leer/modificar]
```

## Estados

| Estado | Cuándo |
|--------|--------|
| Pending | Task creada, aún no especificada completamente |
| Specified | Lista para implementar |
| In Progress | Ejecución en curso |
| Completed | Ejecutada y verificada |
| Blocked | Bloqueada por dependencia o condición externa |
| On Hold | Diferida intencionalmente |
| Obsolete | Ya no aplica |

## Checklist

Antes de finalizar una task, verificar:

1. ¿Cabe en una sesión?
2. ¿Contiene todo el contexto?
3. ¿Los ACs son pass/fail?
4. ¿Declara dependencias con `blocked_by` y path relativo explícito?
5. ¿Lista fuentes de verdad?
6. ¿Preserva invariantes relevantes?
