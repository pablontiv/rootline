---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar rootline tree

**Story**: [S003 Tree and Stats](README.md)

## Contexto

El tree command muestra la jerarquia de documentos como arbol ASCII con conteos de completitud. Respeta la estructura de directorios y lee frontmatter para determinar estado. Reemplaza la logica inline de `/roadmap view`.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline (cobra command)
interfaces:
  - nombre: treeCmd
    metodos:
      - nombre: RunE
        input: "cmd *cobra.Command, args []string"
        output: error
dependencias_externas:
  - github.com/spf13/cobra
tests:
  - tree muestra jerarquia correcta de directorios
  - tree muestra conteos [completados/total] por nivel
  - tree con --output json produce JSON tree
  - tree con path especifico limita scope
```

## Dependencias

- F02 completo (scanner + extractor produce Records con frontmatter)

## Alcance

**In**:
1. Cobra command `tree` con arg opcional: [path] (default: current dir)
2. Scan directorio, agrupar Records por jerarquia de directorios
3. Contar tasks por estado en cada nivel (Story, Feature, Epic)
4. ASCII output con `├──`, `└──`, indentacion
5. Conteos `[n/m]` donde n=completados, m=total
6. JSON output opcional (--output json)

**Out**: Derived state computation (I3), interactive/watch mode

## Estado inicial esperado

- Scanner y extractor funcionales
- Records con frontmatter.estado accesible

## Criterios de Aceptacion

- `rootline tree docs/epics/` muestra arbol ASCII con estructura de directorios
- Cada nodo con tasks muestra `[n/m]` conteo
- `rootline tree docs/epics/ --output json` produce JSON tree structure
- Arbol respeta la jerarquia real del filesystem

## Fuente de verdad

- `src/rootline/docs/intent/v0-rootline.md` seccion 3 (Commands: tree)
