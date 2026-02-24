---
estado: Specified
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T003: Agregar check de frescura de binarios

**Story**: [S003 Binary Bundling](README.md)

## Contexto

Skills cambian muchas veces al día pero releases son menos frecuentes. El workflow no debe re-descargar binarios en cada sync de skills — solo cuando hay un nuevo release. Esto desacopla la frecuencia de sync de skills de la frecuencia de actualización de binarios.

## Alcance

**In**:
1. Comparar versión de binario en marketplace (leer de un VERSION file)
2. Obtener último release tag de rootline (`gh release view --json tagName`)
3. Solo descargar binarios si versión difiere
4. Actualizar VERSION file después de descarga exitosa
5. Log claro: "binaries up-to-date (vX.Y.Z), skipping download" o "new release vX.Y.Z, downloading binaries"

**Out**: Notificaciones de nuevos releases, auto-trigger en release events

## Estado inicial esperado

- T001 completado: descarga de binarios funcional en workflow
- Al menos un release de rootline existe en GitHub

## Criterios de Aceptacion

- Sync de skills sin nuevo release no re-descarga binarios
- Nuevo release trigger descarga de binarios actualizados
- VERSION file en marketplace refleja versión de binarios bundled
- Workflow log indica claramente si descargó o skipeó binarios

## Fuente de verdad

- `.github/workflows/publish-marketplace.yml` (workflow a modificar)
- GitHub Releases API
