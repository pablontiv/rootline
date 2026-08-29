---
tipo: adr
estado: accepted
fecha: '2026-08-28'
contexto: 'La distribución mediante copias desde pre-push mutaba el home sin aprobación ligada al estado y permitía divergencia entre Claude, OpenCode y Pi.'
decision: 'Mantener eliminado post-merge, retirar toda mutación de skills desde pre-push y distribuir el skill canónico mediante rootline skill con symlinks, planes ligados a digest, backups y recibos; los cambios breaking se corrigen desde CHANGELOG.md sin compatibilidad legacy.'
alternativas: 'Conservar copias desde pre-push perpetúa divergencia y mutación implícita; usar scripts o recetas fragmenta el contrato multiplataforma; mantener rutas legacy agrega estado sin valor vigente.'
consecuencias: 'La instalación pasa a ser explícita y reversible; Claude usa ~/.claude/skills y OpenCode/Pi comparten ~/.agents/skills; entornos antiguos requieren remediación deliberada documentada en CHANGELOG.md.'
---
# 0005. Distribuir skills mediante comando explicito

Reemplaza a 0003-eliminar-hook-post-merge-mutante.

## Contexto
La distribución mediante copias desde pre-push mutaba el home sin aprobación ligada al estado y permitía divergencia entre Claude, OpenCode y Pi.

## Decisión
Mantener eliminado post-merge, retirar toda mutación de skills desde pre-push y distribuir el skill canónico mediante rootline skill con symlinks, planes ligados a digest, backups y recibos; los cambios breaking se corrigen desde CHANGELOG.md sin compatibilidad legacy.

## Alternativas descartadas
Conservar copias desde pre-push perpetúa divergencia y mutación implícita; usar scripts o recetas fragmenta el contrato multiplataforma; mantener rutas legacy agrega estado sin valor vigente.

## Consecuencias
La instalación pasa a ser explícita y reversible; Claude usa ~/.claude/skills y OpenCode/Pi comparten ~/.agents/skills; entornos antiguos requieren remediación deliberada documentada en CHANGELOG.md.
