---
tipo: outcome
---
# O15: Coverage feedback loop

El gate de 85% se rompe periódicamente (`9453d5d` bajó threshold → `775a0c5` revirtió; `2581dff` agregó +110 líneas no testeadas en tree.go → CI rojo a 83.7% → `320ea7b` PR de emergencia con 9 archivos de tests). El ciclo se repite porque no hay señal de coverage local: `just test` corre sin `-coverprofile`, no existe `just coverage`, y `.githooks/pre-push` no mide cobertura — la primera alarma siempre es CI fallando.

Este outcome instaura el feedback antes del push: recipe local que mide y reporta, pre-push hook que bloquea regresiones, archivo declarativo de pisos por paquete (todos al estándar de 85, ninguno con descuento), y eliminación del código deprecado/dead que vive en el denominador sin servir al producto.

Resultado observable: cualquier push que baje cobertura total o per-package por debajo de 85% se detiene localmente con mensaje accionable; los desarrolladores no descubren el problema en CI; la cobertura no se "arregla" con tests reactivos post-merge.

Invariantes que las tasks preservan:
- INV1: el threshold no se baja para forzar verde — si un paquete no llega a 85, se sube con tests, no se relaja el gate
- INV2: el piso es 85% uniforme; ningún paquete recibe descuento
- INV3: el feedback ocurre antes del push, no en CI

Scope: el gate vive en el repo de rootline (no se toca `pablontiv/crossbeam@v1`). El threshold global de CI sigue en 85 — este outcome lo refuerza con pre-push + per-package, no lo cambia.
