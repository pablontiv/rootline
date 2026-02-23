---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Agregar seccion links al .stem de epics

**Story**: [S002 Dependency Wiki-Links](README.md)

## Contexto

Rootline ya soporta seccion `links:` en archivos `.stem` (`internal/rules/rules.go:26-43`). Define tipos de links permitidos y patrones de target. El `.stem` de `docs/epics/` no tiene esta seccion configurada. Se necesita agregar para que `rootline graph` sepa que `blocks` es un tipo de link valido apuntando a archivos de task.

El wiki-link `[[blocks:T001-name]]` en el body de un task se parsea como `Link{Type: "blocks", Target: "T001-name"}` por `internal/extract/links.go`. El graph builder crea edges a partir de estos links. Solo falta declarar en el `.stem` que `blocks` es un tipo permitido.

## Dependencias

- Ninguna — primer task de la story

## Alcance

**In**:
1. Agregar seccion `links:` a `docs/epics/.stem`:
   ```yaml
   links:
     allowed: [blocks]
     blocks:
       target: "T*.md"
   ```
2. Verificar con `rootline describe docs/epics/` que la seccion links aparece en el schema efectivo
3. Verificar con `rootline validate` que no rompe archivos existentes

**Out**: No modificar codigo Go. No migrar archivos existentes (eso es T003).

## Estado inicial esperado

- `docs/epics/.stem` tiene secciones `schema:` y `validate:`, pero no `links:`
- `rootline describe docs/epics/` muestra `"links": {}` vacio

## Criterios de Aceptacion

- `docs/epics/.stem` contiene seccion `links:` con `allowed: [blocks]` y `blocks: { target: "T*.md" }`
- `rootline describe docs/epics/ --output json` muestra links con allowed y rules
- `rootline validate --all docs/epics/` no produce errores nuevos
- `go test ./internal/rules/ -race` pasa (no se modifico codigo, pero verificar que el .stem parsea)

## Fuente de verdad

- `docs/epics/.stem` — archivo a modificar
- `internal/rules/rules.go` — LinkSchema struct (referencia, no modificar)
