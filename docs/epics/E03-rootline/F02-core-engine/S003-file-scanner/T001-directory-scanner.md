---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Walk del arbol de directorios con .stemignore

**Story**: [S003 File Scanner](README.md)

## Contexto

El scanner es el paso 1-2 del pipeline: descubre archivos y lee su contenido. Rootline usa su propio archivo `.stemignore` (independiente de `.gitignore`, como `.dockerignore`) para controlar visibilidad. El scanner no filtra por scope — eso es T002.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/index
interfaces:
  - nombre: Scanner
    metodos:
      - nombre: Scan
        input: "rootPath string, registry *extract.Registry"
        output: "[]*extract.Record, error"
dependencias_externas: []
tests:
  - Scan encuentra archivos .md recursivamente
  - Scan excluye archivos en .stemignore
  - Scan excluye directorios .git
  - Scan con directorio vacio retorna lista vacia
  - Scan pasa contenido a extractor y retorna Records
  - .stemignore en subdirectorio aplica a ese subtree
```

## Dependencias

- F02/S002 completado (Extractor interface, Registry, MarkdownExtractor)

## Alcance

**In**:
1. Funcion `Scan(rootPath string, registry *extract.Registry) ([]*extract.Record, error)`
2. Walk recursivo del filesystem desde rootPath
3. Para cada archivo: consultar Registry.ForFile → si extractor existe, leer contenido, Extract()
4. Parsear `.stemignore` files (formato gitignore-style: globs, `#` comentarios, lineas vacias)
5. `.stemignore` por directorio, aplica a ese directorio y subdirectorios
6. Excluir `.git/` directory siempre (hardcoded)

**Out**: Scope matching (T002), query filtering, validation

## Estado inicial esperado

- Registry y MarkdownExtractor funcionales (F02/S002)
- Paquete `internal/index/` existe

## Criterios de Aceptacion

- `Scan("testdir/", registry)` encuentra todos los .md y retorna Records
- Archivos listados en `.stemignore` no aparecen en resultados
- `.stemignore` en subdirectorio excluye archivos solo en ese subtree
- `.git/` directory es excluido
- Archivos sin extractor registrado son ignorados (no error)
- Scanner lee contenido y delega a extractor correctamente

## Fuente de verdad

- `src/rootline/docs/research/I7-extractors-architecture.md` seccion 6 (Pipeline Integration, steps 1-4)
