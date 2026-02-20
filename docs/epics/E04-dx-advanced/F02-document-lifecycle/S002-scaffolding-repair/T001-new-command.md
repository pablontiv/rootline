---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar comando new que scaffold documentos desde schema efectivo

**Story**: [S002 Scaffolding and Repair](README.md)

## Contexto

El comando `rootline new <path>` resuelve el schema efectivo del directorio destino via WalkUp + MergeStemFiles, y genera un archivo Markdown con frontmatter pre-poblado. Los campos required se llenan con defaults (o vacios si no hay default), los campos enum incluyen el primer valor, y se agregan comentarios YAML con valores validos.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: newCmd
    metodos:
      - nombre: RunE
        input: "cmd *cobra.Command, args []string"
        output: "error"
dependencias_externas: []
tests:
  - new genera frontmatter con campos required
  - new aplica defaults del schema
  - new en directorio sin .stem retorna error
  - new no sobreescribe archivo existente
```

## Dependencias

- internal/rules (WalkUp, MergeStemFiles)

## Alcance

**In**:
1. Comando `rootline new <filepath>` (path requerido)
2. Resuelve schema efectivo del directorio padre del filepath
3. Genera archivo .md con frontmatter YAML:
   - Campos required con default value (o vacio)
   - Campos enum con primer valor de values list
   - Comentarios YAML inline con valores validos para enums
4. Body con heading `# ` derivado del filename
5. No sobreescribe si archivo existe (usar `--force`)

**Out**: Templates externos, contenido del body, interactive prompts

## Estado inicial esperado

- internal/rules con WalkUp y MergeStemFiles funcionales
- .stem files existentes para probar

## Criterios de Aceptacion

- `rootline new /tmp/test/doc.md` crea archivo con frontmatter correcto segun .stem del directorio
- Campos required del schema efectivo aparecen en frontmatter
- Campos enum incluyen comentario con valores validos
- `rootline new /tmp/test/existing.md` retorna error "file already exists"
- `rootline validate /tmp/test/doc.md` sobre archivo generado no produce errores de campos required
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `internal/rules/discovery.go` — WalkUp
- `internal/rules/merge.go` — MergeStemFiles
- `internal/rules/rules.go` — StemFile, SchemaField (required, default, values)
