---
estado: Specified
tipo: documentation
ejecutable_en: 1 sesion
---
# T002: Reemplazar tipos hardcodeados con rootline describe

**Story**: [S001 Dynamic Type Discovery](README.md)
**Contribuye a**: task-guide.md no tiene lista hardcodeada de tipos en el template

[[blocks:T001-extract-type-specs]]

## Contexto

task-guide.md tiene tipos hardcodeados en dos lugares: la tabla de Paso 2 (12 tipos con descripcion) y la linea `tipo:` del template de frontmatter. Estos deben reemplazarse con instrucciones para usar `rootline describe` y referencia a type-specs.md.

## Alcance

**In**:
1. Paso 2 (Determinar Tipo): reemplazar tabla hardcodeada con instruccion `rootline describe <story-dir> --field schema.tipo` para descubrir tipos validos del proyecto
2. Template frontmatter: reemplazar linea `tipo: servicio-docker | modulo-sistema | ...` con `tipo: # descubrir via rootline describe <story-dir> --field schema.tipo`
3. Seccion "## Especificacion Tecnica": reemplazar bloques YAML inline con referencia `Ver [type-specs.md](type-specs.md) para templates de especificacion por tipo`
4. Mantener la nota general sobre cuando usar Especificacion Tecnica

**Out**: No eliminar type-specs.md. No modificar .stem files.

## Preserva

- INV1: `rootline describe <story-dir> --field schema.tipo` retorna tipos validos
- Verificar: ejecutar comando en un Story existente, verificar output JSON con values
- INV2: La seccion de ACs y checklist no se modifica
- Verificar: condiciones 1-7 identicas

## Estado inicial esperado

- T001 completado: type-specs.md existe con bloques YAML
- task-guide.md tiene tipos hardcodeados en Paso 2, template, y Especificacion Tecnica

## Criterios de Aceptacion

- Paso 2 menciona `rootline describe <story-dir> --field schema.tipo` como metodo de descubrimiento
- Template frontmatter no tiene lista hardcodeada de tipos
- Seccion Especificacion Tecnica referencia type-specs.md en vez de tener bloques YAML inline
- `rootline describe docs/epics/E04-dx-advanced/F05-dependency-graph/S001-link-extraction/ --field schema.tipo` retorna JSON con values

## Fuente de verdad

- `.claude/skills/roadmap/task-guide.md`
- `.claude/skills/roadmap/type-specs.md`
