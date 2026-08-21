---
tipo: adr
estado: accepted
fecha: "2026-08-20"
contexto: "Schema apply debe validar el estado jerárquico completo y la misma identidad física que su escritura puede reemplazar."
decision: "Validar un StemState prospectivo sobre targets físicos ligados a capabilities abiertas y preservar errores de ancestros externos."
consecuencias: "Ambos report paths compartirán validación y escritura física convergentes; la abstracción general de filesystem queda diferida."
---

# ADR 0001: Evaluar el estado prospectivo completo de stems

## Contexto

`schema apply` consume reportes de propuestas y de análisis. La validación previa incorporada en PR #193 comprueba que cada declaración propuesta sea válida de forma aislada, pero no comprueba que el conjunto final respete la herencia monotónica. Un child stem localmente válido puede ampliar un tipo heredado, ser aplicado con `complete: true` y fallar stem health en la siguiente ejecución de `validate --all`.

La evaluación actual de stem health combina discovery desde disco con reglas de validación. Esa dependencia impide evaluar candidatos virtuales sin escribirlos primero. Ambos report paths también necesitan la misma garantía de durabilidad al persistir `.stem`.

## Decisión

Se separará el discovery del filesystem de la evaluación de reglas. `internal/rules` expondrá un estado lógico de stems y un evaluador puro que use las mismas autoridades de declaración, merge y compatibilidad monotónica que la validación normal.

Cada report path producirá candidatos serializados sin escribir. Todos los candidatos aceptados se superpondrán sobre un único estado prospectivo y se evaluarán como batch. Cualquier diagnóstico de severidad `error` rechazará el plan antes de publicar acciones o escribir archivos; warnings e información no bloquearán.

Las escrituras aprobadas usarán reemplazo atómico por archivo. El batch conservará el contrato best-effort existente: un fallo posterior no revierte archivos aplicados previamente.

Cada target se ligará durante planificación a una capability abierta sobre su padre físico. La evaluación superpondrá esa identidad física y la ejecución renombrará mediante el mismo handle, de modo que un alias symlink interno no pueda hacer que validación y escritura operen sobre stems distintos. La discovery de ancestros externos conservará YAML malformado como `ParseErrors` en vez de abortar mediante un walk-up fail-fast.

## Alternativas descartadas

### Filesystem abstracto inyectable en toda la resolución

Ofrece flexibilidad general, pero exige ampliar `WalkUp`, discovery, parsing, resolución y provenance sin un segundo consumidor real. Se difiere como enhancement #194 con una condición explícita de adopción.

### Validación en un árbol temporal

Reutiliza APIs actuales, pero duplica el corpus, complica symlinks, `.stemignore` y provenance, y puede divergir del filesystem real.

### Escribir, validar y hacer rollback

No garantiza paridad con dry-run, publica temporalmente governance inválido y puede fallar durante el rollback. Contradice la validación previa requerida.

### Transacción multiarchivo

Un journal o rollback global aumenta la complejidad y no garantiza atomicidad real entre renames. El contrato actual best-effort es suficiente si cada archivo individual nunca queda parcial.

### Rechazar todos los symlinks de directorio

Eliminaría la divergencia de identidad de forma simple, pero rompería symlinks internos válidos. Se conserva su soporte ligando validación y escritura a un padre físico abierto y rechazando únicamente escapes o cambios de identidad.

## Consecuencias

- `ValidateStemHealth` conservará su API pública como fachada sobre discovery más evaluación pura.
- Proposal y analyze/inference compartirán planificación, evaluación prospectiva y ejecución.
- `internal/infer` separará transformación YAML de persistencia.
- El envelope de schema apply expondrá diagnósticos prospectivos de stem health.
- Dry-run y ejecución compartirán el mismo veredicto de governance.
- Las escrituras `.stem` preservarán el modo existente y no dejarán contenido parcial.
- Los aliases internos se coalescerán por target físico y no podrán redirigir un write después de planificación.
- Los ancestros gobernantes malformados se publicarán como diagnósticos `yaml-valid` relativos al report root.
- La abstracción general de filesystem no forma parte de este cambio y queda registrada en #194.
