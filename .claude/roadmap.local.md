---
roadmap-root: docs/epics

done-statuses:
  - Completed
  - Obsolete

active-statuses:
  - Pending
  - Specified
  - In Progress

container-types:
  - feature
  - historia

story-close-verify:
  - go test ./... -race
  - go vet ./...
---

# Roadmap — Rootline Project Type Specs

Templates de Especificación Técnica para los tipos usados en este proyecto.
Descubiertos desde `rootline describe <story-dir> --field schema.tipo`.

## software-module

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: # internal/<package>
interfaces:
  - nombre: # nombre de interfaz/tipo a implementar
    metodos:
      - nombre: # nombre del método
        input: # parámetros
        output: # retorno
dependencias_externas:
  - # librería externa (ej: gopkg.in/yaml.v3)
tests:
  - # descripción del test case principal
```

## software-test

```yaml
tipo: software-test
proyecto: rootline
paquete: # internal/<package> a testear
cobertura_objetivo: 85%
tipos_test:
  - unit | integration | e2e
fixtures:
  - # archivos de test data necesarios
```

## ci-cd

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - push | pr | tag | schedule
jobs:
  - nombre: # nombre del job
    pasos:
      - # descripción del paso
artefactos:
  - # binarios, imágenes, paquetes producidos
```

## documentation

No requiere bloque YAML de especificación técnica. Describir el contenido y estructura esperada en prosa dentro de la sección Alcance del task.

## feature / historia

Estos tipos se usan en READMEs de agrupación (Feature e Historia/Story). No llevan Especificación Técnica — su contenido se define por los templates de [epic-guide.md](.claude/skills/roadmap/epic-guide.md) y [story-guide.md](.claude/skills/roadmap/story-guide.md).
