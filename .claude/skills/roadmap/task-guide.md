# Task Guide — Crear Task AI-Ready

## Workflow

### Paso 1: Parsear Argumentos

De `$ARGUMENTS`, extraer:
- **story-path**: ruta a la Story padre (ej: `E01/F13/S001`)
- **task-name**: slug kebab-case (ej: `add-k8s-phase`)
- **descripción**: qué debe hacer el agente AI

### Paso 2: Determinar Tipo de Task

Descubrir los tipos válidos dinámicamente desde el schema del proyecto:

```bash
rootline describe <story-dir> --field schema.tipo
```

Esto retorna los valores de enum válidos para el campo `tipo` en el directorio de la Story. Seleccionar el tipo que mejor describe el trabajo del Task.

Todos los tipos pueden incluir `## Especificacion Tecnica` cuando se beneficien de una spec estructurada. Ver [type-specs.md](type-specs.md) para templates de especificación por tipo, o usar un bloque libre si no hay template predefinido.

### Paso 3: Verificar Story Padre

```bash
# Resolver path real
rootline describe docs/epics/E01-*/F13-*/S001-*/
```

Si no existe → informar al usuario. Sugerir crear con `/roadmap story` primero.

Si existe → leer el README de la Story para:
- Entender la capacidad objetivo
- Extraer contexto relevante para el Task
- Ver Tasks existentes (evitar duplicación)

### Paso 4: Auto-numbering

```bash
# Detectar próximo TXXX en la Story (requiere .stem con id: {type: sequence, prefix: T, digits: 3})
rootline describe <story-dir> --field schema.id.next
```

El comando retorna directamente el próximo identificador (ej: `"T004"`). Requiere que el directorio padre tenga un `.stem` con `id: {type: sequence}` configurado.

### Paso 5: Generar Task File

**5.1**: Crear el archivo con frontmatter correcto usando `rootline new`:

```bash
rootline new <story-dir>/TXXX-task-name.md
```

Esto genera el frontmatter según el `.stem` del directorio, con valores de enum correctos y comentados. El agente edita el contenido del task (contexto, alcance, ACs) pero NO modifica el schema del frontmatter — solo selecciona el valor correcto de cada enum.

**5.2**: Editar el contenido con toda la información necesaria para que un agente AI lo ejecute sin contexto adicional.

**CRÍTICO**: El Task debe ser auto-contenido. Un agente que lea SOLO este archivo debe poder ejecutar el trabajo completo.

### Paso 6: Actualizar Story README

Agregar fila en la tabla "Tasks" del Story README padre:

```markdown
| [TXXX](TXXX-task-name.md) | Descripción breve |
```

**Nota**: NO incluir columna Estado en la tabla. El estado se lee del YAML frontmatter del Task file. `/roadmap view` lo deriva automáticamente.

---

## Template: Task File

**Notas sobre el template**:
- **Wiki-links de dependencia**: Si el Task depende de otro, agregar `[[blocks:TXXX-name]]` debajo del link a la Story. `rootline graph` lee estos links automaticamente para detectar ciclos y resolver orden de ejecucion. Omitir si no hay dependencias.
- **Especificacion Tecnica**: Incluir cuando el Task se beneficie de una especificacion estructurada — aplica a cualquier tipo (IaC, software, ci-cd, etc.). Omitir solo si el Task es puramente textual (ej: documentation sin componente tecnico). Usar el bloque YAML correspondiente al tipo, o un bloque libre si no hay template predefinido.

```markdown
---
estado: Pending
tipo: # descubrir via rootline describe <story-dir> --field schema.tipo
ejecutable_en: 1 sesion
---
# TXXX: [Descripción accionable del task]

**Story**: [SXXX Story Name](README.md)
**Contribuye a**: [criterio específico de la Story]

[[blocks:TXXX-prerequisite-task]]

## Preserva

- INV1: [invariante de la Story a mantener]
  - Verificar: [comando o procedimiento]

## Contexto

[Párrafo breve explicando el contexto necesario. Extraído de la Story padre pero auto-contenido. El agente no necesita leer otro archivo para entender qué hacer.]

## Especificacion Tecnica

Ver [type-specs.md](type-specs.md) para templates de especificación por tipo. Usar el bloque YAML correspondiente al tipo seleccionado, o un bloque libre si no hay template predefinido.

## Dependencias

> Contexto humano complementario. Las dependencias machine-readable se declaran arriba con `[[blocks:TXXX-name]]`.

- [Task/componente que debe existir antes de ejecutar este Task — contexto adicional]
- [Servicio, módulo o config que este Task requiere]

## Alcance

**In**: [Lista específica de lo que el agente DEBE hacer]
1. [Acción concreta 1]
2. [Acción concreta 2]

**Out**: [Lo que NO debe hacer — límites explícitos]

## Estado inicial esperado

- [Prerrequisito 1 que debe existir antes de ejecutar]
- [Prerrequisito 2]

## Criterios de Aceptacion

- [Criterio binario 1 — comando o check específico con resultado esperado]
- [Criterio binario 2 — observable, automático, pass/fail]
- [Criterio binario 3 — sin términos vagos]

## Fuente de verdad

- [Path a código/config que el agente necesita leer/modificar]
- [Path a documentación de referencia]
```

---

## Estados del Task

| Estado | Emoji | Cuándo |
|--------|-------|--------|
| Pending | - | Task creado, sin especificación técnica |
| Specified | 📋 | Especificación técnica completa, listo para implementar |
| In Progress | 🔄 | Ejecución en curso por un agente |
| Completed | ✅ | Ejecutado y verificado exitosamente |
| Blocked | 🚫 | Bloqueado por dependencia externa o task previo |
| On Hold | ⏸️ | Diferido intencionalmente, se retomará después |

---

## Checklist de Validación (7 Condiciones)

Antes de finalizar el Task, verificar mentalmente:

| # | Condición | Pregunta de validación |
|---|-----------|----------------------|
| 1 | Sesión única | ¿Un agente puede completar esto en una sesión? |
| 2 | Sin memoria | ¿El archivo contiene TODO el contexto necesario? |
| 3 | Criterios binarios | ¿Cada criterio es pass/fail sin interpretación? |
| 4 | Verificable | ¿Los criterios referencian comandos/checks reales? |
| 5 | Idempotente | ¿Se puede re-ejecutar sin daño? |
| 6 | I/O explícitos | ¿Estado inicial, resultado y fuentes están declarados? |
| 7 | Contribuye a | ¿Tiene campo "Contribuye a" que traza a un criterio de la Story padre? |

**Nota**: Estas condiciones se validan manualmente al revisar el Task. No hay hooks automáticos configurados actualmente.

---

## Anti-patrones a Evitar

- **Criterio vago**: "Servicio configurado correctamente" → Usar: "`systemctl is-active service` retorna active"
- **Scope inflado**: Task que toca 5 archivos en 3 capas → Dividir en Tasks más pequeños
- **Dependencia implícita**: "Después de configurar X..." → Declarar en "Estado inicial esperado"
- **Sin fuente de verdad**: Agente no sabe qué archivos mirar → Siempre listar paths
