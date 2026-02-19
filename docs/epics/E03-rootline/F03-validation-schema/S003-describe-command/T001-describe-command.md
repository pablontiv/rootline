---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar cobra command rootline describe

**Story**: [S003 Describe Command](README.md)

## Contexto

El describe command es la ventana de debuggability de Rootline. Muestra el schema efectivo resultado del merge de todos los .stem ancestros. Cada campo incluye source (cual .stem lo definio). Este comando reemplaza la necesidad de leer .stem files manualmente y elimina el hardcoding de valores validos en hooks/skills.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline (cobra command)
interfaces:
  - nombre: describeCmd
    metodos:
      - nombre: RunE
        input: "cmd *cobra.Command, args []string"
        output: error
dependencias_externas:
  - github.com/spf13/cobra
tests:
  - describe con directorio valido retorna JSON contract completo
  - describe con --field extrae dot-path
  - describe sin .stem ancestors retorna schema vacio
  - applies array muestra orden correcto de .stem files
```

## Dependencias

- F02/S001 (walk-up discovery + merge = effective schema)

## Alcance

**In**:
1. Cobra command `describe` con arg: <path> (directorio)
2. Walk-up discovery + merge → effective StemFile
3. Output JSON contract de I5: version, kind, path, applies, scope, schema, validate, derive, state, links
4. Cada campo en schema incluye `source` (path del .stem)
5. `applies` array: lista ordenada de .stem files mergeados
6. `--field` flag para dot-path extraction (ej: `--field schema.Tipo.values`)
7. Sin .stem ancestors → schema vacio (no error)

**Out**: Explain tracing (I4 deferred), derive/state processing

## Estado inicial esperado

- Walk-up discovery y merge funcionales (F02/S001)
- Cobra skeleton con describe stub

## Criterios de Aceptacion

- `rootline describe docs/prd/` produce JSON con schema completo y source en cada campo
- `rootline describe docs/prd/ --field schema.Tipo.values` retorna array de valores
- `rootline describe docs/prd/ --field applies` retorna lista de .stem paths
- `rootline describe path/sin/stem/` retorna `{"schema":{},"applies":[],...}` sin error
- JSON output incluye `"version": 1` y `"kind": "rootline/describe"`

## Fuente de verdad

- `src/rootline/docs/research/I5-describe-contract.md` seccion 2 (Describe JSON Contract)
- `src/rootline/docs/research/I5-describe-contract.md` seccion 3.6 (Effective Schema Comparison)
