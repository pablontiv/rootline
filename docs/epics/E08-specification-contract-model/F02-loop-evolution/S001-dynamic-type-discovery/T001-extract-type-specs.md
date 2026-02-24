---
estado: Specified
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Extraer templates de especificacion tecnica a type-specs.md

**Story**: [S001 Dynamic Type Discovery](README.md)
**Contribuye a**: type-specs.md existe con templates YAML extraidos de task-guide.md

## Contexto

task-guide.md contiene 7 bloques YAML de especificacion tecnica (servicio-docker, modulo-sistema, operacion-sistema, software-module, software-test, ci-cd) inline. Estos son proyecto-especificos y acoplan el skill a un proyecto particular.

## Alcance

**In**:
1. Crear `.claude/skills/roadmap/type-specs.md` con todos los bloques YAML de spec tecnica extraidos de task-guide.md
2. Agregar header explicativo: "Templates de especificacion tecnica por tipo. Estos son templates de referencia del proyecto actual — adaptar segun .stem del proyecto."
3. Mantener estructura: titulo por tipo + bloque YAML

**Out**: No modificar task-guide.md aun (eso es T002). No crear nuevos tipos.

## Preserva

- INV1: Los bloques YAML son copia exacta de los actuales en task-guide.md
- Verificar: diff entre bloques en task-guide.md y type-specs.md es identico

## Estado inicial esperado

- task-guide.md tiene seccion "## Especificacion Tecnica" con bloques YAML para cada tipo
- No existe `.claude/skills/roadmap/type-specs.md`

## Criterios de Aceptacion

- Archivo `.claude/skills/roadmap/type-specs.md` existe
- Contiene todos los bloques YAML de spec tecnica que estan en task-guide.md
- Tiene header explicativo sobre adaptabilidad por proyecto
- Bloques son copia exacta (no modificados)

## Fuente de verdad

- `.claude/skills/roadmap/task-guide.md` (fuente)
- `.claude/skills/roadmap/type-specs.md` (destino)
