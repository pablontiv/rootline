---
estado: Specified
tipo: task
---
# T004: Define compatibility expectations between rootline CLI versions and the Pi extension.

**Outcome**: [O01 Map Rootline integration surface](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T003-classify-pi-exposure.md]]

## Preserva

- INV1: No Pi tool is implemented before its Rootline command contract and risk class are documented.
  - Verificar: Check downstream tasks reference this Outcome as source of truth.

## Contexto

Esta task forma parte de O01 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Define compatibility expectations between rootline CLI versions and the Pi extension.

## Alcance

**In**:
1. A compatibility policy exists for CLI version detection and unsupported commands.
2. The policy defines failure messages for missing or too-old rootline binaries.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T003-classify-pi-exposure.md` está completada o su salida está disponible.

## Criterios de Aceptación

- A compatibility policy exists for CLI version detection and unsupported commands.
- The policy defines failure messages for missing or too-old rootline binaries.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `install.sh`
- `.goreleaser.yml`
- `README.md`
