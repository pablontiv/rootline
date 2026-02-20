---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar cobra command rootline query

**Story**: [S002 Query Command](README.md)

## Contexto

El query command es la interfaz principal de Rootline para buscar documentos. Combina scanner (file discovery) + extractor (produce Records) + query engine (filtra). Los flags mapean directamente al JSON query contract de I1.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline (cobra command)
interfaces:
  - nombre: queryCmd
    metodos:
      - nombre: RunE
        input: "cmd *cobra.Command, args []string"
        output: error
dependencias_externas:
  - github.com/spf13/cobra
tests:
  - query con --where filtra correctamente
  - query con multiples --where combina con AND
  - query con --count retorna numero
  - query con --limit restringe resultados
  - query con --field extrae dot-path
  - query con --from limita scope de busqueda
  - query sin --where retorna todos los Records
```

## Dependencias

- F04/S001 (query engine funcional)
- F02 completo (scanner + extractor)

## Alcance

**In**:
1. Cobra command `query` con flags: --where (repeatable), --from, --field, --count, --limit
2. Parse de --where string: `'field op value'` → Condition struct
3. Multiples --where = implicit AND
4. --from limita root path del scan (default: current dir)
5. --field para dot-path extraction del resultado
6. --count para retornar CountResult en vez de rows
7. JSON output por defecto, table output opcional (--output table)

**Out**: Cursor/pagination, custom output formats

## Estado inicial esperado

- Query engine funcional (F04/S001)
- Scanner y extractor funcionales (F02)
- Cobra skeleton con query stub (F01)

## Criterios de Aceptacion

- `rootline query --where 'estado eq Pending'` retorna JSON rows
- `rootline query --where 'tipo eq servicio-docker' --where 'estado eq Pending'` combina filtros
- `rootline query --where 'estado eq Completado' --count` retorna JSON count
- `rootline query --where 'estado eq Pending' --limit 5` retorna max 5 rows
- `rootline query --where 'estado eq Pending' --field path` retorna solo paths
- `rootline query --from docs/prd/` limita busqueda a docs/prd/

## Fuente de verdad

- `src/rootline/docs/research/I1-query-operators.md` seccion 5 (CLI Flag Mapping)
- `src/rootline/docs/research/I1-query-operators.md` seccion 6 (Field Extraction)
