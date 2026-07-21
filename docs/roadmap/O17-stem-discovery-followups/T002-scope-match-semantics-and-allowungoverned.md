---
estado: Pending
tipo: task
---
# T002: Definir la semántica de `scope.match` en todo el CLI y resolver `AllowUngoverned`

**Outcome**: [O17 Consolidar deuda de stem-native-discovery](README.md)
**Contribuye a**: INV1 (verificar contra el binario) e INV2 (no debilitar lo entregado).

## Preserva

- La distinción gobernado/bootstrap ya entregada: los comandos que mutan exigen esquema resuelto; los de inferencia (`schema propose`, `analyze`) derivan uno y no pueden exigirlo.

## Contexto

`scope.match` es honrado por sólo 5 de 11 comandos, y el reparto es casi el inverso del principio. Pasan `WithScopeResolver`: `validate`, `fix`, `tree`, `analyze`, `schema`. NO lo pasan: `query`, `graph`, `explain`, `stats`, `init`, `migrate`. Verificado por ejecución: con un `.stem` de `scope.match: "T*.md"`, `validate --all` ve 1 registro y `query` ve 2. El mismo árbol tiene dos definiciones de "qué es un registro" según el comando.

La exploración completa está en el cambio SDD `decouple-inference-scan` (persistida en Engram, topic `sdd/decouple-inference-scan/explore`; el archivo openspec está gitignoreado). Su recomendación: primero decidir qué significa `scope.match` en todo el CLI y hacerlo consistente, y recién después desacoplar los comandos de inferencia.

`index.AllowUngoverned()` es un parche provisional que hoy mantiene vivos a `schema propose` y `analyze` sobre árboles sin esquema. Dos escenarios del spec de `stem-native-discovery` fueron marcados SUPERSEDED por él. Su destino se decide aquí: si los comandos de inferencia sueltan el resolver y se apoyan en `.stemignore`, `AllowUngoverned` se puede borrar.

## Criterios de aceptación

- Una decisión escrita de qué significa `scope.match`: "archivos que son registros" (filtro de descubrimiento) vs "registros que se validan" (filtro de gobierno), y qué conjunto de comandos lo aplica.
- Todos los comandos consistentes con esa decisión; `query`/`stats` dejan de discrepar con `validate`/`tree`.
- Decisión explícita sobre `AllowUngoverned`: eliminarlo o formalizarlo, y actualizar o restaurar los dos escenarios del spec marcados SUPERSEDED en consecuencia.
- Si `query`/`stats` empiezan a honrar `scope`, documentar el cambio de comportamiento (breaking pre-1.0).
