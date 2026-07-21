---
estado: Pending
tipo: task
---
# T001: Corregir la escritura de `init` jerárquico que pisa todos los niveles

**Outcome**: [O17 Consolidar deuda de stem-native-discovery](README.md)
**Contribuye a**: INV1 (verificar contra el binario) e INV2 (no debilitar lo entregado).

## Preserva

- El marcado root-only ya correcto de `runInitHierarchical`: sólo `stemMap["."]` recibe `root: true`. Esta tarea NO cambia qué se marca, sino dónde se escribe cada nivel.

## Contexto

`cmd/rootline/init.go:155-163` recorre `stemMap` (raíz `.` y cada nivel hijo como `E01`, `E01/F01`) y escribe TODAS las entradas al mismo path `filepath.Join(absTarget, ".stem")`, ignorando la clave del mapa. `internal/infer/schema_gen.go:79` `GenerateHierarchicalSchema` sí devuelve un mapa multinivel (falla con "need at least 2 levels" si no). Resultado: un `init` genuinamente jerárquico escribe todos los niveles al `.stem` raíz, pisándose en orden de iteración aleatorio; los `.stem` por nivel nunca se escriben en sus directorios.

El bug viene de `669547a` ("hierarchical .stem generation"), muy anterior a `stem-native-discovery`. No fue introducido por el marcador.

`TestInitHierarchicalMarkerRootOnly` (`cmd/rootline/init_test.go`) tampoco lo atrapa: su fixture `E01/F01` colapsa a un solo stem, así que no ejercita el caso multinivel ni afirma nada sobre el path de escritura ni sobre los hijos sin marcar.

## Criterios de aceptación

- Cada entrada de `stemMap` se escribe a `filepath.Join(absTarget, <clave>, ".stem")`, respetando la clave del mapa.
- Un test que genere una jerarquía real de ≥2 niveles y verifique: (a) se escribe un `.stem` por nivel en su directorio, (b) sólo el raíz declara `root: true`, (c) los hijos no lo declaran.
- `just check` y `just test` en verde. Verificación adicional ejecutando `rootline init` sobre un árbol jerárquico real y listando los `.stem` producidos.
