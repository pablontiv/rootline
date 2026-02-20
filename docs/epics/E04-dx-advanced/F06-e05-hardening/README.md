# F06: E05 Hardening

**Epic**: [E04](../README.md)
**Objetivo**: Rootline CLI maneja escenarios reales descubiertos en E05 (validacion contra 114 archivos del homeserver) con UX coherente
**Beneficio**: Elimina 3 gaps de usabilidad encontrados al usar rootline contra datos no controlados: fix sin soporte de directorios, describe sin guidance, init sin warnings para estructuras mixtas
**Milestone**: `rootline fix --all` repara directorios completos, `rootline describe` sugiere `init` cuando no hay .stem, `rootline init` advierte sobre contenido mixto

## Scope

**In**: fix --all con batch output, describe hints para schema vacio, init mixed-content warnings
**Out**: Nuevos comandos, cambios a la logica core de validacion, inferencia de required fields

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Directory-wide Fix](S001-directory-wide-fix/) | fix opera sobre directorios completos igual que validate --all |
| S002 | [Guidance UX](S002-guidance-ux/) | describe e init guian al usuario cuando encuentran escenarios vacios o mixtos |

## Dependencias

- E04/F02 (Document Lifecycle) completado — fix, init, new ya implementados
- E05 completado — findings documentados

## Fuente de verdad

- `cmd/rootline/fix.go` — fix command
- `cmd/rootline/describe.go` — describe command
- `cmd/rootline/init.go` — init command
- `cmd/rootline/validate.go` — referencia para --all pattern
