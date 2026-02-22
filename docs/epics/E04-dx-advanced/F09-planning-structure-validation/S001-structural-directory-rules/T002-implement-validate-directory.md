---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar ValidateDirectory con require_index y min_children

**Story**: [S001 Structural Directory Rules](README.md)

[[blocks:T001-extend-stemfile-structural-types]]

## Contexto

Con T001 completado, `StemFile` tiene un campo `Structural` con `SubdirRules`. Ahora se necesita una funcion que reciba un directorio y un `StemFile` efectivo, y valide las reglas estructurales contra los subdirectorios reales del filesystem.

La funcion opera sobre el filesystem directamente (no sobre Records extraidos), porque las reglas son sobre la estructura de directorios, no sobre contenido de archivos.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: ValidateDirectory
    metodos:
      - nombre: ValidateDirectory
        input: "dir string, stem *StemFile"
        output: "[]ValidationError"
dependencias_externas: []
tests:
  - Directorio con 3 subdirs y require_index README.md — 1 subdir sin README produce 1 error
  - Directorio con 1 subdir y min_children 2 produce error
  - Directorio con 5 subdirs y max_children 3 produce error
  - Sin structural rules — retorna slice vacio
  - Severity warn produce ValidationError con Severity warn
```

## Dependencias

- T001 completado (StructuralRules structs disponibles en StemFile)

## Alcance

**In**:
1. Crear `internal/rules/structural.go` con funcion `ValidateDirectory(dir string, stem *StemFile) []ValidationError`
2. Implementar require_index: listar subdirs, verificar que cada uno tiene el archivo indicado
3. Implementar min_children / max_children: contar subdirs inmediatos
4. Respetar severity del SubdirRules (default: "error")
5. Tests unitarios con directorios temporales (t.TempDir)

**Out**: No integrar con el comando validate (eso es T003). No validar archivos, solo estructura.

## Estado inicial esperado

- T001 completado: `StemFile.Structural.Subdirs` tiene campos RequireIndex, MinChildren, MaxChildren, Severity
- `internal/rules/structural.go` no existe

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestValidateDirectory` pasa
- Directorio con subdir sin README.md + require_index "README.md" → ValidationError con Rule "require_index"
- Directorio con 1 subdir + min_children 2 → ValidationError con Rule "min_children"
- Directorio con 5 subdirs + max_children 3 → ValidationError con Rule "max_children"
- StemFile sin structural rules → []ValidationError vacio (len 0)
- ValidationError.Source contiene path del .stem file

## Fuente de verdad

- `internal/rules/rules.go` — StructuralRules, SubdirRules structs (de T001)
- `internal/rules/validate.go` — ValidationError struct (reusar)
