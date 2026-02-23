---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar migration log

**Story**: [S002 Bulk Operations](README.md)

[[blocks:T001-migrate-rename]]

## Contexto

Cada operacion de `rootline migrate` debe registrarse para auditoria y trazabilidad. El log es un archivo append-only de JSON Lines en la raiz del repositorio.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/migrate
interfaces:
  - nombre: MigrationLog
    metodos:
      - nombre: Append
        input: "entry MigrationEntry"
        output: error
      - nombre: Read
        input: ""
        output: "[]MigrationEntry, error"
dependencias_externas: []
tests:
  - Append crea archivo si no existe
  - Append agrega linea a archivo existente
  - Read parsea todas las entradas
  - Entry tiene timestamp, type, details, files_affected
```

## Dependencias

- T001 completado (rename operation genera entries de log)

## Alcance

**In**:
1. Archivo `.rootline-migrations` en raiz del repositorio (JSON Lines)
2. MigrationEntry struct: Timestamp, Type (rename|...), Details map, FilesAffected int
3. Append: serializar entry como JSON, append con newline
4. Read: parsear archivo linea por linea
5. rootline migrate commands appendean al log despues de aplicar cambios (no en dry-run)

**Out**: Log viewer command, log rotation, remote log, compression

## Estado inicial esperado

- rootline migrate --rename funcional (T001)

## Criterios de Aceptacion

- `rootline migrate --rename` appenda entry al log
- `.rootline-migrations` es JSON Lines valido (una entrada por linea)
- Cada entry tiene timestamp ISO 8601, type, details, files_affected
- `--dry-run` NO escribe al log
- `go test ./internal/migrate/ -run TestLog -v` pasa

## Fuente de verdad

- `internal/migrate/` (package de S001)
