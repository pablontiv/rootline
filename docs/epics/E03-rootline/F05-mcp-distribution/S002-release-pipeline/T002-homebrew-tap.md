---
estado: Completed
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T002: Script de instalacion curl|bash

**Story**: [S002 Release Pipeline](README.md)

[[blocks:T001-goreleaser-config]]

## Contexto

Se evaluo Homebrew tap (requiere repo adicional + PAT) vs script de instalacion curl|bash (zero dependencias, funciona en cualquier Linux/macOS). Se eligio el script por simplicidad y menor mantenimiento.

## Implementacion

Script `install.sh` en la raiz del repo. Instalacion:

```bash
curl -fsSL https://raw.githubusercontent.com/pablontiv/rootline/master/install.sh | bash
```

El script:
- Detecta OS (linux/darwin) y arquitectura (amd64/arm64)
- Obtiene la ultima version desde GitHub API
- Descarga el binario correcto desde GitHub Releases
- Instala en `~/.local/bin` o `/usr/local/bin`
- Soporta curl y wget
- Variable `ROOTLINE_INSTALL_DIR` para directorio custom

## Criterios de Aceptacion

- `shellcheck install.sh` pasa sin errores
- Script detecta OS y arquitectura correctamente
- Descarga e instala el binario de la ultima release
- `rootline --version` funciona post-instalacion

## Fuente de verdad

