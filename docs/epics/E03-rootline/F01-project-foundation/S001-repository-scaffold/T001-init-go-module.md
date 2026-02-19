---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Inicializar Go module con estructura de directorios

**Story**: [S001 Repository Scaffold](README.md)

## Contexto

Rootline es un CLI Go standalone. El intent document (v0-rootline.md) define la arquitectura con paquetes separados para extract, rules, index, query, mcp, y cli. Este Task crea el Go module y la estructura de directorios base sin implementacion.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: / (root module)
interfaces: []
dependencias_externas:
  - github.com/spf13/cobra
  - github.com/spf13/viper
  - gopkg.in/yaml.v3
  - github.com/stretchr/testify
tests:
  - go build ./... compila sin errores
```

## Dependencias

- D11 (GitHub org/user) resuelto — necesario para `go mod init github.com/ORG/rootline`

## Alcance

**In**:
1. `go mod init` con module path correcto
2. Crear directorios: `cmd/rootline/`, `internal/extract/`, `internal/rules/`, `internal/index/`, `internal/query/`, `internal/mcp/`
3. Crear `cmd/rootline/main.go` minimal (package main, func main)
4. `go mod tidy` con dependencias base
5. Crear `.gitignore` para Go project

**Out**: Implementacion de cobra commands (es T002), CI/CD, tests

## Estado inicial esperado

- Repositorio Git existe (puede ser el monorepo homeserver o repo standalone)
- Go instalado (1.22+)
- D11 resuelto (org/user para module path)

## Criterios de Aceptacion

- `go build ./cmd/rootline/` produce binario sin errores
- `ls internal/` muestra 5 directorios (extract, rules, index, query, mcp)
- `go.mod` tiene module path correcto y dependencias declaradas
- `.gitignore` incluye patrones Go estandar

## Fuente de verdad

- `src/rootline/docs/intent/v0-rootline.md` seccion 3 (Architecture)
- `src/rootline/docs/intent/v0-rootline.md` seccion 7 (Stack Summary)
