# Story Guide — Crear Contrato Semántico

## Workflow

### Paso 1: Parsear Argumentos

De `$ARGUMENTS`, extraer:
- **feature-path**: ruta al Feature padre (ej: `E01/F13`)
- **story-name**: slug kebab-case (ej: `layer3-validation`)
- **capacidad**: descripción de la capacidad entregada

### Paso 2: Verificar Feature Padre

```bash
# Verificar que el Feature padre existe
rootline describe <feature-dir>
```

Si no existe → informar al usuario. Sugerir crear con `/roadmap epic` primero.

Si existe → leer el README del Feature para entender contexto y scope.

### Paso 3: Auto-numbering

```bash
# Detectar próximo SXXX en el Feature (requiere .stem con id: {type: sequence, prefix: S, digits: 3})
rootline describe <feature-dir> --field schema.id.next
```

El comando retorna directamente el próximo identificador (ej: `"S003"`). Requiere que el directorio padre tenga un `.stem` con `id: {type: sequence}` configurado.

### Paso 4: Generar Story

**4.1**: Crear directorio y generar README con frontmatter correcto:

```bash
mkdir -p <feature-dir>/SXXX-story-name/
rootline new <feature-dir>/SXXX-story-name/README.md
```

Esto genera el frontmatter según el `.stem` heredado. El agente edita el contenido (antes/después, ACs) pero NO modifica el schema del frontmatter.

**4.2**: Editar el contenido del README con estructura antes/después.

### Paso 5: Actualizar Feature README

Agregar fila en la tabla "Stories" del Feature README padre:

```markdown
| [SXXX](SXXX-story-name/) | Capacidad entregada |
```

---

## Template: Story README

```markdown
# SXXX: [Nombre descriptivo de la capacidad]

**Feature**: [FXX Feature Name](../README.md)
**Capacidad**: [una línea describiendo qué capacidad nueva existe]
**Cubre**: [aspecto del milestone del Feature que esta Story cubre]

## Antes / Despues

**Antes**: [Descripción del estado actual. Qué NO se puede hacer hoy. Qué riesgo existe.]

**Despues**: [Descripción del estado objetivo. Qué capacidad nueva tiene el sistema. Qué riesgo se elimina.]

## Criterios de Aceptacion (semanticos)

> Los ACs deben ser verificables y cada uno debe ser cubierto por al menos un Task (via 'Contribuye a')

- [ ] [Capacidad observable 1 — usuario verifica]
- [ ] [Capacidad observable 2 — usuario verifica]

## Invariantes

- INV1: [propiedad a preservar]
  - Verificar: [comando o procedimiento]
- INV2: [propiedad a preservar]
  - Verificar: [comando o procedimiento]

## Tasks

| Task | Descripcion | Estado |
|------|-------------|--------|
| (se llenan con `/roadmap task`) | | |

## Fuente de verdad

- [path a código/config relevante]
- [path a documentación relevante]
```

---

## Notas

- Los criterios de aceptación de una Story son **semánticos** (de capacidad), no operativos
- Las Tasks se crean después con `/roadmap task` y se agregan a la tabla automáticamente
- Una Story NO está pensada para ejecutarse en una sola sesión de AI — es un agregador de Tasks
- El "Antes/Después" es la pieza más importante: define el valor entregado

---

## Guía para Antes/Después por Dominio

El Antes/Después debe ser concreto, no genérico. Ejemplos por dominio:

### IaC / Infrastructure
```markdown
**Antes**: No existe backup automatizado. Riesgo de pérdida de datos ante fallo de disco.
**Despues**: Kopia ejecuta backups cada 6h con retención 30d. Restore verificado con test.
```

### Software Development
```markdown
**Antes**: No existe parser de frontmatter. 7 consumidores usan regex independientes con 4 patrones distintos.
**Despues**: MarkdownExtractor produce Records JSON con frontmatter parseado. Un solo punto de parsing.
```

### CI/CD / Distribution
```markdown
**Antes**: Builds son manuales. No hay artefactos publicados ni versionamiento.
**Despues**: GitHub Actions produce binarios multi-plataforma en cada tag. Disponible via Homebrew.
```

### Governance / Framework
```markdown
**Antes**: Reglas de validación están dispersas en 6 skills con regex frágiles.
**Despues**: .stem files definen schemas. `rootline validate` reemplaza todas las validaciones.
```

**Anti-patrón**: "Antes: no existe. Después: existe." — Demasiado genérico, no comunica valor.
