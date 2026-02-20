---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar flag --staged en validate para archivos en staging area

**Story**: [S002 Git Integration](README.md)

## Contexto

Para que el pre-commit hook sea eficiente, `rootline validate` necesita un flag `--staged` que solo valide archivos .md en la staging area de git (obtenidos via `git diff --cached --name-only`). Esto evita validar todo el repositorio en cada commit.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: validateCmd (extended)
    metodos:
      - nombre: getStagedFiles
        input: ""
        output: "[]string, error"
dependencias_externas: []
tests:
  - --staged obtiene lista de archivos via git diff --cached
  - --staged filtra solo archivos .md
  - --staged sin archivos staged retorna exit code 0
  - --staged requiere --all (no funciona con archivo especifico)
```

## Dependencias

- T001 (hooks command, para testing end-to-end)
- Comando validate existente

## Alcance

**In**:
1. Flag `--staged` en validate command
2. Ejecutar `git diff --cached --name-only` para obtener archivos staged
3. Filtrar a solo archivos .md (o archivos que tengan extractor registrado)
4. Pasar lista filtrada a la logica de validacion existente (como si fueran argumentos de archivo)
5. Si no hay archivos .md staged, exit code 0 sin output

**Out**: --unstaged flag, diff-based validation, pre-push integration

## Estado inicial esperado

- Comando validate funcional con modo archivo y --all
- Git repository con staging area

## Criterios de Aceptacion

- `rootline validate --staged --all` sin archivos staged retorna exit code 0
- `rootline validate --staged --all` con .md staged valida solo esos archivos
- `rootline validate --staged --all` con .md staged invalido retorna exit code 1
- Archivos no-.md en staging area son ignorados
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `cmd/rootline/validate.go` — validate command (modificar)
- git diff --cached --name-only — mecanismo de discovery
