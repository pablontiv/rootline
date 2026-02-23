---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T003: Reescribir post-aprobacion — solo crear .md + validate + enable model invocation

**Story**: [S002 Skill Rootline Integration](README.md)

## Contexto

El skill tiene tres problemas de scope en su flujo post-aprobacion:

1. **Implementa tasks**: Despues de aprobar un plan, el skill intenta ejecutar el trabajo descrito en los tasks (crear servicios, escribir codigo, etc.). Solo deberia crear los archivos .md de planificacion. `/roadmap loop` es el unico subcomando que implementa.

2. **Sin validacion**: Los documentos creados no se validan contra el schema .stem. rootline validate deberia ejecutarse despues de cada Write.

3. **Model invocation bloqueada**: `disable-model-invocation: true` en el frontmatter impide que Claude invoque el skill programaticamente. Ya se elimino este campo — este task documenta y verifica el cambio.

## Dependencias

- S001 completado (validate necesita .stem completo para ser util)

## Alcance

**In**:
1. Reescribir "Fase 2: Ejecucion (post-aprobacion)" para que explicitamente diga "solo crear archivos .md"
2. Reescribir "Paso 6: Crear Estructura (post-aprobacion)" para incluir `rootline validate` despues de cada Write
3. Verificar que `disable-model-invocation` no existe en frontmatter
4. Actualizar seccion `/roadmap loop` Fase 1 Discovery para usar rootline query en vez del script Python

**Out**: Cambios a la logica de /roadmap loop Fase 3 (ejecucion de tasks), cambios a los guide files, cambios al framework

## Estado inicial esperado

- `.claude/skills/roadmap/SKILL.md` tiene Fase 2 que menciona "Crear archivos/directorios" pero no prohibe implementar
- `disable-model-invocation: true` ya fue eliminado del frontmatter
- `/roadmap loop` Fase 1 referencia el script Python de pending

## Criterios de Aceptacion

- Seccion "Fase 2" del SKILL.md dice explicitamente "solo crear archivos .md" o "NO implementar tasks"
- Paso 6 incluye `rootline validate <path>` despues de crear cada archivo
- Paso 6 incluye `rootline fix <path>` como fallback si validacion falla
- No existe `disable-model-invocation` en el frontmatter del SKILL.md
- Seccion `/roadmap loop` Fase 1 usa `rootline query` en vez de "script Python de `/roadmap pending`"

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md` — archivo a modificar (Fase 2, Paso 6, frontmatter, loop Fase 1)
