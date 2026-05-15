---
estado: Completed
tipo: task
---
# T006: Remove `domain:` from rootline engine

**Outcome**: [O14 Field-agnostic refactor](README.md)
**Contribuye a**: rootline no tiene concepto de "domain" — tipo y comportamiento se declaran explícitamente en el .stem

[[blocked_by:./T004-refactor-tree-frontmatter-map.md]]
[[blocked_by:./T005-remove-title-magic-from-query.md]]

## Preserva

- INV1: Los .stems que usaban `domain: identifier` deben migrar a `type: sequence` explícito
  - Verificar: grep de `domain:` en stems conocidos; si existen, documentar migration path
- INV2: `go test ./...` verde después de eliminar el código
  - Verificar: `cd /home/shared/rootline && go test ./...`

## Contexto

`domain:` era un mecanismo de alias semántico donde `domain: identifier` se traducía automáticamente a `type: sequence`. Esto mezcla la declaración del schema con conocimiento de dominio.

Los consumers en tree.go (T004) y query.go (T005) ya no dependen de domain lookup. Con esos eliminados, se puede remover `domain:` del engine sin romper nada.

`ResolveDomainType()` era el único con comportamiento real. Al eliminarlo, cualquier .stem que use `domain: identifier` debe declarar `type: sequence` explícitamente. Documentar esta migración.

## Alcance

**In**:
1. Eliminar `SchemaField.Domain` del struct en `internal/rules/rules.go`
2. Eliminar `FindFieldByDomain()`, `DomainAliases()`, `coreDomains` map
3. Eliminar `ResolveDomainType()` (o marcarlo como eliminated, borrando la lógica)
4. Eliminar referencias a `domain:` en `internal/rules/validate.go` si las hay
5. Actualizar cualquier .stem del repo que use `domain:` para usar `type:` explícito
6. Agregar nota de migración en CHANGELOG o comentario de commit si algún .stem externo usaba domain:

**Out**:
- No tocar archivos fuera de `internal/rules/` y los .stems del repo
- No cambiar `type: section` todavía (eso es T007)

## Estado inicial esperado

- T004 y T005 completadas (ningún consumer en tree.go/query.go usa domain lookup)
- `go test ./...` pasa
- `FindFieldByDomain`, `DomainAliases`, `coreDomains`, `ResolveDomainType` existen en el código

## Criterios de Aceptación

- `SchemaField` no tiene campo `Domain`
- `FindFieldByDomain()`, `DomainAliases()`, `coreDomains`, `ResolveDomainType()` eliminados
- `go test ./...` verde
- Ningún .stem en el repo usa `domain:` sin `type:` explícito

## Fuente de verdad

- `internal/rules/rules.go` — SchemaField, domain functions
- `internal/rules/domains.go` (si existe como archivo separado)
- `internal/rules/validate.go`
