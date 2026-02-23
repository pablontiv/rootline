---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Add ScanBodyFields to internal/extract

**Story**: [S001 Body Text Scanning](README.md)

## Contexto

Muchos archivos markdown legacy tienen metadata en el body usando formato bold-colon (`**Estado**: Completada`, `**Tipo**: lxc`) en vez de YAML frontmatter. El proposal engine necesita detectar estos patrones para sugerir extraccion a frontmatter en vez de reportar "required field missing" sin contexto.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/extract
interfaces:
  - nombre: ScanBodyFields
    metodos:
      - nombre: ScanBodyFields
        input: "body string"
        output: "map[string]string"
dependencias_externas: []
tests:
  - "**Estado**: Completada" retorna {"estado": "Completada"}
  - "**Key With Spaces**: value" retorna {"key with spaces": "value"}
  - Body sin patrones retorna mapa vacio
  - Multiples patrones en lineas distintas retorna todos
  - Patron con parentesis "**Estado**: Completada (PRD-era)" captura valor completo
```

## Alcance

**In**:
1. Agregar funcion `ScanBodyFields(body string) map[string]string` en `internal/extract/extract.go`
2. Regex: `\*\*([^*]+)\*\*:\s*(.+)` — captura key (sin asteriscos) y value (resto de linea)
3. Keys normalizadas a lowercase, values preservan case original
4. Agregar tests en `internal/extract/extract_test.go`

**Out**: No modificar `MarkdownExtractor.Extract()` — `ScanBodyFields` es helper standalone, no parte del pipeline de extraccion. Sera consumido por el proposal engine en S002.

## Estado inicial esperado

- `internal/extract/extract.go` existe con `MarkdownExtractor`
- `internal/extract/extract_test.go` tiene tests existentes como patron
- `go test ./internal/extract/` pasa sin errores

## Criterios de Aceptacion

- `go test ./internal/extract/ -run TestScanBodyFields -v` pasa con todos los casos
- `ScanBodyFields("**Estado**: Completada\n**Tipo**: lxc")` retorna `{"estado": "Completada", "tipo": "lxc"}`
- `ScanBodyFields("**Estado**: Completada (PRD-era, sin Tasks)")` retorna `{"estado": "Completada (PRD-era, sin Tasks)"}`
- `ScanBodyFields("no bold patterns here")` retorna mapa vacio (len == 0)
- `go vet ./internal/extract/` sin errores

## Fuente de verdad

- `internal/extract/extract.go` — archivo a modificar
- `internal/extract/extract_test.go` — tests a extender
