---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Crear script de instalación de binarios

**Story**: [S003 Binary Bundling](README.md)

## Contexto

El consumidor del marketplace necesita una forma fácil de instalar el binario de rootline desde los archivos bundled. Un install script que detecte OS/arch automáticamente y copie el binario correcto a un directorio en PATH.

## Alcance

**In**:
1. `install.sh` en raíz del marketplace — shell script POSIX-compatible
2. Detectar OS (linux/darwin) y arquitectura (amd64/arm64)
3. Copiar binario correcto desde `bin/{os}-{arch}/rootline`
4. Destino: `~/.local/bin/rootline` (crear directorio si no existe)
5. Verificar que el binario funciona (`rootline --version`)
6. Mensaje claro si la plataforma no está soportada

**Out**: Instalación automática post-skill-install, Windows support

## Estado inicial esperado

- T001 completado: binarios existen en `bin/`

## Criterios de Aceptacion

- `./install.sh` detecta OS y arch correctamente en linux y macOS
- Copia binario al destino y lo hace ejecutable
- `rootline --version` funciona después de instalación
- Falla gracefully si plataforma no soportada con mensaje claro
- Script es POSIX-compatible (no requiere bash)

## Fuente de verdad

- Patrones comunes de install scripts (Homebrew, rustup, etc.)
- `bin/` directory structure definida en T001
