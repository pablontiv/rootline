---
estado: Specified
tipo: documentation
ejecutable_en: 1 sesion
---
# T003: Crear subagente sdd-validator

**Story**: [S002 Workflow Integration](README.md)
**Contribuye a**: Existe subagente sdd-validator en .claude/agents/

## Contexto

La cadena de trazabilidad (Epic.Postcondiciones → Feature.Satisface → Story.Cubre → Task.Contribuye_a) y la propagacion de invariantes necesitan validacion automatica post-materialization. Un subagente especializado puede verificar gaps sin cargar el context principal.

## Alcance

**In**:
1. Crear archivo `.claude/agents/sdd-validator.md` con frontmatter: name, description, tools (Read, Bash, Grep, Glob), model (haiku)
2. System prompt que verifica: cada Epic tiene Postcondiciones, cada Feature declara Satisface, cada Story declara Cubre, cada Task declara Contribuye_a
3. System prompt que verifica propagacion de invariantes: Epic.Invariantes → Feature.Invariantes → Story.Invariantes → Task.Preserva
4. Output: reporte de gaps con archivos que necesitan correccion

**Out**: No modificar SKILL.md (eso es T002 o tarea futura). No crear hooks.

## Preserva

- INV1: No se modifican archivos existentes
- Verificar: solo se crea un archivo nuevo

## Estado inicial esperado

- `.claude/agents/` existe como directorio
- No existe `.claude/agents/sdd-validator.md`

## Criterios de Aceptacion

- Archivo `.claude/agents/sdd-validator.md` existe
- Tiene frontmatter con name: sdd-validator, tools: Read/Bash/Grep/Glob
- System prompt menciona verificacion de Postcondiciones/Satisface/Cubre/Contribuye_a
- System prompt menciona verificacion de propagacion de Invariantes/Preserva

## Fuente de verdad

- `.claude/agents/sdd-validator.md` (nuevo)
