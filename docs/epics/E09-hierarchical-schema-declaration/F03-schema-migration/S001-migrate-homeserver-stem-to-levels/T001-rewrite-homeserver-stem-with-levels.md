---
estado: Specified
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Rewrite homeserver epics .stem with levels and validate

**Story**: [S001 Migrate homeserver .stem to levels](README.md)
**Contribuye a**: homeserver/docs/epics usa un solo `.stem` con schema diferenciado por nivel

## Preserva

- INV5: Todos los documentos existentes siguen validando correctamente
  - Verificar: `rootline validate --all /opt/homeserver/automation/docs/epics/`

## Contexto

The homeserver project at `/opt/homeserver/automation/docs/epics/` has a single flat `.stem` file that applies the same schema to all 293 documents across 4 hierarchy levels. Analysis showed:
- Epics: only `estado` used, `tipo` rarely present
- Features: `tipo` in ~16% of docs
- Stories: `tipo` in ~25%
- Tasks: `tipo` in ~89%, `ejecutable_en` in 100%

The `.stem` should be rewritten with `levels:` to enforce appropriate schemas per level. The current flat schema fields (`estado`, `tipo`, `ejecutable_en`, `cliente`) should be distributed to the appropriate levels.

Current `.stem` content:
```yaml
version: 1
schema:
  id: { type: sequence, prefix: E, digits: 2 }
  estado: { type: enum, values: [...], required: true }
  tipo: { type: enum, values: [...], severity: warn }
  cliente: { type: string }
  ejecutable_en: { type: string }
```

Target structure:
```yaml
version: 1
schema:
  estado: { type: enum, required: true, values: [...] }
levels:
  epic:
    match: "E*"
    children: [feature]
    schema:
      id: { type: sequence, prefix: E, digits: 2 }
  feature:
    match: "F*"
    children: [story]
    schema:
      id: { type: sequence, prefix: F, digits: 2 }
      tipo: { type: enum, severity: warn, values: [...] }
  story:
    match: "S*"
    children: [task]
    schema:
      id: { type: sequence, prefix: S, digits: 3 }
  task:
    match: "T*"
    children: []
    schema:
      id: { type: sequence, prefix: T, digits: 3 }
      tipo: { type: enum, required: true, values: [...] }
      ejecutable_en: { type: string, severity: error }
```

## Dependencias

- F01 + F02: The levels engine and caller migration must be complete before migrating real files

## Alcance

**In**:
1. Rewrite `/opt/homeserver/automation/docs/epics/.stem` with `levels:` section
2. Distribute fields to appropriate levels based on data analysis
3. Keep `estado` as base schema (all levels)
4. Run `rootline validate --all` to verify no regressions
5. Fix any documents that fail validation due to stricter per-level enforcement

**Out**: Rootline's own `.stem` migration (S002), engine changes

## Estado inicial esperado

- F01 and F02 are complete — levels engine works end-to-end
- `/opt/homeserver/automation/docs/epics/.stem` exists with flat schema
- 293 documents across 4 levels

## Criterios de Aceptacion

- `/opt/homeserver/automation/docs/epics/.stem` has `levels:` section with 4 levels
- `rootline validate --all /opt/homeserver/automation/docs/epics/` passes
- Tasks have `tipo` and `ejecutable_en` enforced
- Features have `tipo` as warn severity
- Base `estado` applies to all levels

## Fuente de verdad

- `/opt/homeserver/automation/docs/epics/.stem` — file to rewrite
