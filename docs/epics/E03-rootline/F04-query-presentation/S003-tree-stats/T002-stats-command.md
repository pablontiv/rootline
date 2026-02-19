---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar rootline stats

**Story**: [S003 Tree and Stats](README.md)

## Contexto

El stats command produce un resumen de conteos agrupados por tipo y/o estado. Es la respuesta rapida a "cuantos tasks pendientes hay?" o "cuantos servicios docker tenemos?". Complementa query (filtrado) y tree (jerarquia) con agregacion.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline (cobra command)
interfaces:
  - nombre: statsCmd
    metodos:
      - nombre: RunE
        input: "cmd *cobra.Command, args []string"
        output: error
dependencias_externas:
  - github.com/spf13/cobra
tests:
  - stats muestra conteos por estado
  - stats muestra conteos por tipo
  - stats con --from limita scope
  - stats produce JSON con version y kind
```

## Dependencias

- F02 completo (scanner + extractor)

## Alcance

**In**:
1. Cobra command `stats` con flag: --from (scope)
2. Scan y extract Records
3. Agrupar por frontmatter.estado y frontmatter.tipo
4. JSON output: `{version:1, kind:"rootline/stats", by_estado:{...}, by_tipo:{...}, total:N}`
5. Table output opcional

**Out**: Custom grouping fields, time-series stats

## Estado inicial esperado

- Scanner y extractor funcionales

## Criterios de Aceptacion

- `rootline stats` produce JSON con conteos por estado y tipo
- `rootline stats --from docs/epics/` limita scope
- JSON output incluye version:1 y kind:"rootline/stats"
- Conteos son correctos (verificar contra grep manual)

## Fuente de verdad

- `src/rootline/docs/intent/v0-rootline.md` seccion 3 (Commands: stats)
