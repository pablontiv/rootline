---
estado: Completed
tipo: task
---
# T009: Fix `renderChild()` — remove hardcoded "estado" from ASCII display

**Outcome**: [O14 Field-agnostic refactor](README.md)
**Contribuye a**: la capa de presentación visual del árbol usa el nombre de campo del schema, no "estado" hardcodeado

## Preserva

- INV1: El output JSON de `rootline tree` no cambia (Frontmatter map ya es correcto)
  - Verificar: `rootline tree /home/shared/rootline/docs/roadmap --output json | python3 -c "import json,sys; d=json.load(sys.stdin); assert d.get('version') == 2"`
- INV2: `go test ./...` verde
  - Verificar: `cd /home/shared/rootline && go test ./...`

## Contexto

T004 y T008 removieron correctamente los campos hardcodeados del **engine y la estructura de datos**: `treeNode.Estado` fue eliminado, `treeNode.Frontmatter map[string]any` lo reemplaza, y `buildTree` popula el mapa correctamente. El output JSON es field-agnostic.

Sin embargo, la función `renderChild()` en `cmd/rootline/tree.go` (~línea 195) todavía hardcodea `"estado"` para el **ASCII rendering**:

```go
if e, ok := node.Frontmatter["estado"]; ok {
    estado = fmt.Sprintf("%v", e)
}
```

Y el help text del comando (línea ~21) dice "derived from frontmatter.estado fields".

Estos son los únicos remanentes de "estado" como nombre de campo hardcodeado en el comando tree.

## Alcance

**In**:
1. `renderChild()`: reemplazar `node.Frontmatter["estado"]` por el nombre de campo de lifecycle desde el schema activo. Si el schema no está disponible en ese contexto, usar el primer campo de tipo enum que el stem defina, o simplemente omitir el display de estado en el ASCII si no hay forma limpia de resolverlo.
2. Help text del comando: actualizar para no mencionar "estado" como nombre de campo específico.
3. Cubrir con test o verificar con `rootline tree /home/shared/rootline/docs/roadmap` que el ASCII rendering sigue mostrando el estado correctamente.

**Out**:
- No cambiar el output JSON (ya es correcto)
- No cambiar la lógica de agregación ni los contadores
- No refactorizar `renderChild()` más allá de lo necesario

## Criterios de Aceptación

- `cmd/rootline/tree.go` no contiene el string literal `"estado"` en ninguna rama de código
- Help text del comando tree no menciona "estado" como nombre de campo
- `rootline tree /home/shared/rootline/docs/roadmap` muestra el estado en el ASCII rendering sin errores
- `go test ./...` verde

## Fuente de verdad

- `cmd/rootline/tree.go` — función `renderChild()` y help text del comando
