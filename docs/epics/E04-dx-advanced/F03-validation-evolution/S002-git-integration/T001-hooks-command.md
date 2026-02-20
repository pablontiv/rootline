---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar comando hooks con install/uninstall/status

**Story**: [S002 Git Integration](README.md)

## Contexto

El comando `rootline hooks` gestiona git hooks para validacion automatica. `install` escribe un script bash en `.git/hooks/pre-commit` que ejecuta `rootline validate --staged --all`. `uninstall` lo remueve. `status` reporta si esta instalado.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: hooksCmd
    metodos:
      - nombre: RunE (install)
        input: "cmd *cobra.Command, args []string"
        output: "error"
      - nombre: RunE (uninstall)
        input: "cmd *cobra.Command, args []string"
        output: "error"
      - nombre: RunE (status)
        input: "cmd *cobra.Command, args []string"
        output: "error"
dependencias_externas: []
tests:
  - hooks install crea archivo .git/hooks/pre-commit
  - hooks install no sobreescribe hook existente sin --force
  - hooks uninstall remueve hook creado por rootline
  - hooks status reporta instalado/no instalado
```

## Dependencias

- Repositorio git inicializado (.git/ existe)

## Alcance

**In**:
1. Subcomando `rootline hooks install` — escribe pre-commit script
2. Subcomando `rootline hooks uninstall` — remueve pre-commit si fue creado por rootline
3. Subcomando `rootline hooks status` — reporta si hook esta instalado
4. Script pre-commit: `#!/bin/sh\nrootline validate --staged --all`
5. Marker en script para identificar hooks de rootline (comentario `# rootline-managed`)
6. No sobreescribir hook existente no-rootline sin `--force`

**Out**: Pre-push hooks, commit-msg hooks, hook templates customizables

## Estado inicial esperado

- Repositorio git con .git/hooks/ existente
- rootline binary en PATH

## Criterios de Aceptacion

- `rootline hooks install` crea `.git/hooks/pre-commit` con contenido ejecutable
- `.git/hooks/pre-commit` contiene `# rootline-managed` marker
- `rootline hooks status` reporta "installed" despues de install
- `rootline hooks uninstall` remueve el hook
- `rootline hooks status` reporta "not installed" despues de uninstall
- `rootline hooks install` con hook existente no-rootline retorna error sin --force
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `cmd/rootline/root.go` — rootCmd para agregar subcomando
- `.git/hooks/` — directorio target
