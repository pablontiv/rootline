---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Merge type-driven de lista ordenada de StemFiles

**Story**: [S001 Stem Parser and Merge](README.md)

## Contexto

Dados N archivos .stem ordenados root-to-leaf, el merge produce un StemFile efectivo. El comportamiento de merge es determinado por el tipo YAML de cada valor: maps merge recursivamente, arrays reemplazan, scalars reemplazan, null remueve. Este modelo es universal — no hay logica especifica por seccion.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: Merge
    metodos:
      - nombre: MergeStemFiles
        input: "entries []StemEntry"
        output: "*StemFile"
dependencias_externas: []
tests:
  - Map merge (hijo agrega y overridea keys)
  - Array replace (hijo reemplaza completamente)
  - Scalar replace
  - Null removes inherited key
  - Merge de 3+ niveles (abuelo, padre, hijo)
  - Merge de lista vacia retorna StemFile vacio
```

## Dependencias

- T001 (StemFile struct) y T002 (WalkUp produces StemEntry list)

## Alcance

**In**:
1. Funcion `MergeStemFiles(entries []StemEntry) *StemFile`
2. Type-driven merge rules:
   - map + map → key-level recursive merge
   - array + array → child replaces parent
   - scalar + scalar → child replaces parent
   - any + null → key removed
3. Source tracking: cada campo en el resultado incluye el path del .stem que lo definio/overrideo
4. Tests para los 4 merge examples de I5 seccion 1.3

**Out**: Describe command (F03), validate against effective schema

## Estado inicial esperado

- T001 y T002 completados
- StemFile y StemEntry disponibles

## Criterios de Aceptacion

- `MergeStemFiles([parent{a:1,b:2}, child{b:3,c:4}])` produce `{a:1,b:3,c:4}`
- `MergeStemFiles` con array en child reemplaza array de parent completamente
- `MergeStemFiles` con null en child remueve key de parent
- `MergeStemFiles` de 3 niveles aplica merge secuencial correctamente
- Source tracking indica cual .stem definio cada campo
- `MergeStemFiles([])` retorna StemFile vacio sin error

## Fuente de verdad

- `src/rootline/docs/research/I5-describe-contract.md` seccion 1.3 (Type-Driven Merge Strategy)
- `src/rootline/docs/research/I5-describe-contract.md` seccion 4 (Edge Cases)
