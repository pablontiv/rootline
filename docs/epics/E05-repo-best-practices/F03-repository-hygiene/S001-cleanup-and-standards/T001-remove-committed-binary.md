---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Eliminar binario rootline del tracking git

**Story**: [S001 Cleanup and Standards](README.md)

## Contexto

El binario compilado `rootline` (8.8MB) está committeado al repositorio pese a que `.gitignore` lista `/rootline`. Esto infla el tamaño del repo innecesariamente. El binario se genera con `go build ./cmd/rootline/` y no debería estar versionado — cada developer lo compila localmente y goreleaser produce los binarios de release.

## Dependencias

- Ninguna

## Alcance

**In**:
1. Ejecutar `git rm --cached rootline` para eliminar del tracking sin borrar el archivo local
2. Verificar que `.gitignore` ya contiene `/rootline` (no necesita cambios)
3. Commit del cambio

**Out**: Reescribir historia de git (BFG/filter-branch) para eliminar el binario de commits anteriores — esto es destructivo y cambia SHAs

## Estado inicial esperado

- `rootline` (binario ELF, ~8.8MB) está tracked por git
- `.gitignore` contiene `/rootline`
- `git ls-files rootline` retorna `rootline`

## Criterios de Aceptacion

- `git ls-files rootline` no retorna nada (archivo fuera del tracking)
- El archivo `rootline` sigue existiendo localmente (no borrado)
- `.gitignore` contiene `/rootline` (sin cambios necesarios)
- `git status` no muestra `rootline` como untracked (porque .gitignore lo cubre)

## Fuente de verdad

- `rootline` — binario a eliminar del tracking
- `.gitignore` — ya tiene la entrada correcta
