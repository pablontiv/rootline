---
tipo: adr
estado: accepted
fecha: "2026-08-28"
contexto: "La distribución de skills requiere flujos aprobados de desinstalación y restauración que no dependan de estado mutable entre planificación y aplicación."
decision: "Las aprobaciones de desinstalación y restauración se vinculan al recibo seleccionado, a la evidencia actual de cada destino soportado y, en restauración, a las preimágenes registradas."
consecuencias: "La restauración detecta deriva mediante un nuevo digest antes de mutar, solo reemplaza enlaces instalados observados y verifica los respaldos antes de planificar."
---

## Contexto

Los flujos de recuperación de `skilldist` deben operar sobre destinos soportados fijos y recibos emitidos por instalaciones previas. La aprobación debe ser stateless: una aplicación solo puede continuar si el digest del plan vigente coincide con el digest aprobado.

## Decisión

La desinstalación se planifica desde el último recibo completo de instalación y exige que cada destino soportado siga siendo un enlace simbólico intacto y coherente con la evidencia del recibo.

La restauración se planifica desde el recibo solicitado, verifica antes de planificar todos los respaldos necesarios para preimágenes no ausentes, e incluye en el digest el ID del recibo, la evidencia actual de los destinos y las preimágenes registradas.

## Alternativas descartadas

- Restaurar automáticamente directorios antiguos durante desinstalación: descartado porque mezclaría desinstalación con recuperación de preimágenes y podría sobrescribir estado no observado.
- Omitir la evidencia actual del digest de restauración: descartado porque permitiría aplicar una aprobación stale sobre un destino que cambió después de la planificación.
- Confiar solo en rutas grabadas en el recibo: descartado porque los destinos soportados deben provenir del conjunto fijo actual.

## Consecuencias

- Los cambios externos en destinos gestionados bloquean o invalidan la aprobación antes de mutar.
- La restauración registra sus propios respaldos y recibo, lo que permite auditar el estado que reemplazó.
- Una restauración puede requerir una nueva aprobación si el contenido del origen receipted deriva aunque el enlace siga apuntando al mismo origen.
