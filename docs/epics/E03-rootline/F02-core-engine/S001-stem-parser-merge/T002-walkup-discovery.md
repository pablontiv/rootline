---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Walk-up desde target path hasta .git recolectando .stem files

**Story**: [S001 Stem Parser and Merge](README.md)

## Contexto

Rootline resuelve configuracion usando walk-up discovery: desde el target path, camina hacia arriba recolectando archivos .stem hasta encontrar el boundary del repositorio (.git). Los archivos se retornan ordenados de root a leaf para el merge top-down posterior.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: Discovery
    metodos:
      - nombre: WalkUp
        input: "targetPath string"
        output: "[]StemEntry, error"
dependencias_externas: []
tests:
  - Walk-up encuentra .stem en multiples niveles
  - Walk-up se detiene en .git
  - Walk-up con directorios sin .stem los salta
  - Walk-up sin ningun .stem retorna lista vacia
  - Walk-up sin .git se detiene en filesystem root
```

## Dependencias

- T001 completado (StemFile struct y ParseStem disponibles)

## Alcance

**In**:
1. Struct `StemEntry` con Path (relative) y parsed `StemFile`
2. Funcion `WalkUp(targetPath string) ([]StemEntry, error)`
3. Walk hacia arriba desde targetPath
4. Stop signal: directorio contiene `.git/`
5. Retornar ordenado root-to-leaf (invertir el orden de discovery)
6. Tests con filesystem temporal (t.TempDir())

**Out**: Merge algorithm (T003), scope matching

## Estado inicial esperado

- T001 completado: `ParseStem` disponible en `internal/rules/`
- Paquete compilable

## Criterios de Aceptacion

- `WalkUp("a/b/c/d/")` con .stem en `a/` y `a/b/c/` retorna `[a/.stem, a/b/c/.stem]` (root-to-leaf)
- `WalkUp` se detiene al encontrar `.git/`
- `WalkUp` en path sin .stem files retorna `[]StemEntry{}` vacio, sin error
- `WalkUp` en path sin .git sube hasta filesystem root
- Tests usan t.TempDir() (no dependen de filesystem real)

## Fuente de verdad

- `src/rootline/docs/research/I5-describe-contract.md` seccion 1.1 (Two-Phase Resolution)
- `src/rootline/docs/research/I5-describe-contract.md` seccion 1.2 (Stop Signal)
