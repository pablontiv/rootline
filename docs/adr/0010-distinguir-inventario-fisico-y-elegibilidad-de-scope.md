---
tipo: adr
estado: accepted
fecha: '2026-08-29'
contexto: 'La revisión adversarial demostró que un descendiente excluido por .stemignore podía satisfacer el scope ancestro porque StemState no distinguía presencia física de pertenencia gobernada.'
decision: 'Conservar todas las entradas físicas en StemState, marcar su elegibilidad mediante StemStateEntry.Ignored con la semántica existente de .stemignore y excluir las ignoradas al atribuir scope.match por Chain.'
alternativas: 'Se descartó pasar records de Scan porque acoplaría la evaluación pura al scanner; se descartó eliminar entradas del snapshot porque perdería inventario físico útil para overlays; se descartó importar internal/index porque crearía un ciclo de dependencias.'
consecuencias: 'Stem health comparte la pertenencia observable del scanner sin perder pureza ni inventario; la detección añade una comprobación de ignore por archivo y exige mantener sincronizada la semántica duplicada de .stemignore.'
---
# 0010. Distinguir inventario fisico y elegibilidad de scope

Reemplaza a 0009-resolver-scope-matches-mediante-cadena-de-gobierno.

## Contexto
La revisión adversarial demostró que un descendiente excluido por .stemignore podía satisfacer el scope ancestro porque StemState no distinguía presencia física de pertenencia gobernada.

## Decisión
Conservar todas las entradas físicas en StemState, marcar su elegibilidad mediante StemStateEntry.Ignored con la semántica existente de .stemignore y excluir las ignoradas al atribuir scope.match por Chain.

## Alternativas descartadas
Se descartó pasar records de Scan porque acoplaría la evaluación pura al scanner; se descartó eliminar entradas del snapshot porque perdería inventario físico útil para overlays; se descartó importar internal/index porque crearía un ciclo de dependencias.

## Consecuencias
Stem health comparte la pertenencia observable del scanner sin perder pureza ni inventario; la detección añade una comprobación de ignore por archivo y exige mantener sincronizada la semántica duplicada de .stemignore.
