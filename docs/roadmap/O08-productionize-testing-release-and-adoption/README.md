---
estado: Pending
tipo: outcome
---
# O08: Productionize testing, release, and adoption

## Objetivo

Make the Pi Rootline integration reliable enough to publish and adopt: test matrix, CI, security posture, docs, and optional Rootline CLI onboarding helper.

## Criterios de Éxito

- CE1: Package validation runs in CI against representative Rootline projects.
  - Verificar: CI shows passing tests for extension package and Rootline fixtures.
- CE2: Users have a documented stable install path and onboarding flow.
  - Verificar: Docs show local, git, and npm/package install flows plus troubleshooting.

## Invariantes

- INV1: Release automation must not publish unvalidated extension code or include unintended files.
  - Verificar: Review package files, CI, and npm/git release config.

## Alcance

**In**:
- CI/test matrix
- Security review
- Release path
- Adoption docs
- rootline pi init decision/implementation

**Out**:
- Changing Rootline data model
- Adding non-Pi editor integrations

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-add-extension-ci-validation.md) | Add CI validation for the Pi package. |
| [T002](T002-add-headless-pi-smoke-tests.md) | Add headless Pi smoke tests for package discovery and core workflows. |
| [T003](T003-define-package-release-strategy.md) | Define release strategy for integrations/pi versus separate package. |
| [T004](T004-harden-package-security.md) | Review package security and supply-chain posture. |
| [T005](T005-write-user-adoption-docs.md) | Write user-facing adoption and troubleshooting docs. |
| [T006](T006-design-or-implement-rootline-pi-init.md) | Design or implement rootline pi init onboarding helper. |
