---
estado: Completed
tipo: task
---
# T002: Add CHANGELOG.md

**Contribuye a**: give rootline consumers a human-readable record of breaking changes and notable additions across releases.

## Alcance

**In**:
- Create `/CHANGELOG.md` following the backscroll format (sections: [Unreleased], then versioned entries with Fixed/Features/Documentation/CI/CD headings)
- Seed with the current pre-1.0 history extracted from `git log` (major milestones only, not every commit)

**Out**:
- CHANGELOG is maintained manually on release, not auto-generated

## Criterios de Aceptación

- `test -f /home/shared/rootline/CHANGELOG.md` passes
- Contains an `[Unreleased]` section and at least one versioned entry (e.g. latest release)
- Format consistent with `/home/shared/backscroll/CHANGELOG.md`
- `git -C /home/shared/rootline log --oneline -1` shows a conventional commit

## Fuente de verdad

- /home/shared/rootline/CHANGELOG.md (new)
- /home/shared/backscroll/CHANGELOG.md (format reference)
