# Marco Roadmap — Modelo Simple AI-Native

## Propósito

El roadmap organiza trabajo para agentes AI con contexto limitado. Cada unidad debe ser comprensible desde el estado actual del repositorio, ejecutable sin memoria histórica y verificable con criterios binarios.

## Jerarquía canónica

Máximo dos niveles:

```text
Outcome/Objetivo   ← opcional, agrupa una intención común
└── Task           ← unidad ejecutable por un agente en una sesión
```

Para trabajo pequeño, usar solo tasks:

```text
Task
```

## Outcome

Un **Outcome** existe solo cuando ayuda a agrupar varias tasks bajo un resultado observable.

Crear un Outcome cuando:

- hay más de 5 tasks relacionadas, o
- varias tasks comparten una misma capacidad/objetivo, o
- se necesita declarar invariantes o criterios de éxito comunes.

No crear Outcome cuando:

- hay 1–5 tasks independientes y autoexplicativas,
- el agrupador solo repite el nombre de las tasks,
- hace falta inventar taxonomía para justificarlo.

Un Outcome debe declarar:

- objetivo observable,
- criterios de éxito verificables,
- invariantes que sus tasks preservan,
- límites explícitos de alcance.

## Task

La **Task** es el átomo ejecutable. Una task válida cumple todas estas condiciones:

1. Ejecutable en una sola sesión.
2. No depende de memoria histórica.
3. Contiene todo el contexto necesario.
4. Tiene criterios de aceptación binarios.
5. Es verificable con comandos/checks concretos.
6. Tiene input/output y alcance explícitos.
7. Declara dependencias con links machine-readable.
8. Es idempotente o descartable.

Si no cumple una condición, dividirla o convertirla en Outcome + tasks.

## Dependencias

La convención canónica es:

```markdown
[[blocked_by:./T001-setup.md]]
```

Semántica:

```text
Task actual depende de ./T001-setup.md.
./T001-setup.md desbloquea la task actual.
```

Para dependencias entre Outcomes, usar path relativo explícito:

```markdown
[[blocked_by:../O01-setup/T001-setup.md]]
```

No usar targets bare como `[[blocked_by:T001-setup]]`: rootline solo puede resolverlos por basename único y se rompen si hay duplicados.

## Criterios de aceptación

Los ACs de una task deben ser:

- observables,
- automáticos o procedimentales sin ambigüedad,
- pass/fail,
- verificables desde el estado actual.

Ejemplos válidos:

- `go test ./...` retorna exit 0.
- `rootline validate --all docs/roadmap/` retorna exit 0.
- El archivo `docs/foo.md` existe y contiene la sección `## Uso`.

Ejemplos inválidos:

- “queda bien integrado”,
- “funciona correctamente”,
- “mejorar DX”.

## Trazabilidad simple

- Toda Task dentro de un Outcome declara a qué criterio del Outcome contribuye.
- Toda Task sin Outcome declara directamente el resultado esperado.
- Las invariantes del Outcome se copian o referencian en la sección `Preserva` de cada Task.

## Nombres

Usar vocabulario del dominio de implementación, no de investigación ni clasificación interna.

- ❌ `T001-validar-hipotesis-cap-07`
- ✅ `T001-validar-frontmatter-yaml`

## Escala

- 1–5 tasks: normalmente no hace falta Outcome.
- 6–20 tasks relacionadas: 1 Outcome + tasks.
- Múltiples objetivos independientes: varios Outcomes.
- Si un Outcome supera ~20 tasks, dividir por objetivos observables, no por capas artificiales.
