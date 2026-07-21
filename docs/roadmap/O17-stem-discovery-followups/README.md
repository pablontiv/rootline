---
tipo: outcome
estado: Pending
---
# O17: Consolidar la deuda descubierta durante stem-native-discovery

El cambio `stem-native-discovery` reemplazó el límite `.git` del descubrimiento de esquemas por un marcador `root: true` en `.stem`. Al implementarlo y verificar el binario comando por comando aparecieron dos problemas reales que quedaron fuera de su alcance: uno es un bug preexistente de `init` jerárquico, el otro es una inconsistencia transversal en cómo los comandos aplican `scope.match`. Ninguno tenía un lugar durable fuera de la memoria de trabajo; este outcome los hace de primera clase.

Este outcome NO reabre `stem-native-discovery`. Registra el trabajo diferido para que no dependa de recordarlo.

Invariantes:
- INV1: cada tarea se verifica contra el binario real, no sólo contra la suite. Ambos defectos originales sobrevivieron a tests verdes y sólo se detectaron ejecutando el CLI.
- INV2: ningún cambio de esta deuda debilita el preflight de límite ni el contrato de `WalkUp` ya entregados.

Scope: corregir el bug de escritura de `init` jerárquico y definir la semántica de `scope.match` en todo el CLI (incluida la suerte de `AllowUngoverned`). No incluye la reposición del README (`reposition-rootline`) ni la decisión de push/autoupdate del release.
