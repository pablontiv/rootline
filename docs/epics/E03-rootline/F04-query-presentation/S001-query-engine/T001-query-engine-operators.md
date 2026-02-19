---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar 5 operadores + and + count + limit

**Story**: [S001 Query Engine](README.md)

## Contexto

Los 5 operadores fueron derivados de analisis de 18 consumidores reales (I1). Cada operador mapea a al menos un patron de codigo existente. No se incluyeron operadores sin evidencia de uso (gt, lt, or, not, startswith, endswith). El query engine filtra una lista de Records contra un WHERE clause declarativo.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/query
interfaces:
  - nombre: QueryEngine
    metodos:
      - nombre: Execute
        input: "records []*extract.Record, query *Query"
        output: "*QueryResult, error"
dependencias_externas: []
tests:
  - eq filtra por valor exacto
  - ne excluye por valor
  - in filtra por lista de valores
  - contains busca substring en campo
  - contains en body busca en contenido del documento
  - exists verifica presencia de campo
  - and combina multiples condiciones
  - count retorna numero en vez de rows
  - limit restringe cantidad de resultados
  - Campo inexistente en eq no matchea (no error)
  - Campo inexistente en ne matchea
  - Array field con eq matchea si contiene el valor
```

## Dependencias

- F02/S002 (Record type con Frontmatter y Body)

## Alcance

**In**:
1. Struct `Query` con From, Select, Where, Limit, Count, Cursor
2. Struct `Condition` representando operadores (eq, ne, in, contains, exists, and)
3. Funcion `Execute(records []*Record, query *Query) (*QueryResult, error)`
4. Null handling: eq contra campo inexistente = no match, ne contra campo inexistente = match
5. Array fields: eq matchea si array contiene valor, in matchea interseccion
6. `QueryResult` con Version, Kind, Meta (count, next_cursor), Rows
7. Count mode: retorna `CountResult` con count numerico

**Out**: CLI command wiring, field shortcuts (T002), order_by

## Estado inicial esperado

- Record type disponible (internal/extract/)
- Paquete internal/query/ existe

## Criterios de Aceptacion

- `Execute(records, {where: eq("estado", "Pending")})` retorna solo Records con estado=Pending
- `Execute(records, {where: in("tipo", ["lxc","vm"])})` retorna Records con tipo lxc o vm
- `Execute(records, {where: contains("body", "migration")})` busca en Body
- `Execute(records, {where: and(eq("tipo","servicio-docker"), eq("estado","Pending"))})` combina
- `Execute(records, {count: true})` retorna CountResult con numero
- `Execute(records, {limit: 5})` retorna maximo 5 rows
- QueryResult JSON incluye version:1 y kind:"rootline/query"

## Fuente de verdad

- `src/rootline/docs/research/I1-query-operators.md` seccion 3 (Operator Spec)
- `src/rootline/docs/research/I1-query-operators.md` seccion 4 (JSON Query Contract)
- `src/rootline/docs/research/I1-query-operators.md` seccion 9 (Edge Cases)
