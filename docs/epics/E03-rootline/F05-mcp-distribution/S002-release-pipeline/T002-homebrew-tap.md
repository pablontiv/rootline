---
estado: Pending
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T002: Crear Homebrew tap y formula

**Story**: [S002 Release Pipeline](README.md)

## Contexto

Homebrew es el package manager dominante en macOS y popular en Linux. Un tap dedicado permite `brew install org/tap/rootline`. goreleaser puede generar la formula automaticamente en cada release.

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - tag (via goreleaser)
jobs:
  - nombre: homebrew-update
    pasos:
      - goreleaser genera formula
      - Push a tap repository
artefactos:
  - Formula/rootline.rb en tap repo
```

## Dependencias

- T001 completado (goreleaser funcional)
- D11 resuelto (GitHub org/user para tap repo)

## Alcance

**In**:
1. Crear repositorio `homebrew-tap` (o agregar a goreleaser brews config)
2. goreleaser brews section que genera formula automaticamente
3. Formula con desc, homepage, url (release asset), sha256
4. Test: `brew install org/tap/rootline && rootline --version`

**Out**: Homebrew core formula (requiere popularidad), Linux package managers

## Estado inicial esperado

- goreleaser funcional (T001)
- GitHub repository para tap

## Criterios de Aceptacion

- `.goreleaser.yaml` tiene seccion `brews` configurada
- Tap repository existe con Formula/ directory
- `goreleaser --snapshot` genera formula .rb
- Formula incluye desc, homepage, install section correctos
- `brew install --build-from-source Formula/rootline.rb` instala correctamente (test local)

## Fuente de verdad

