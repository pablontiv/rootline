---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implement Migrate Value Detector

**Story**: [S002 Proposal Analysis Engine](README.md)

[[blocks:T001-create-proposal-types-and-basic-detectors]]

## Contexto

Algunos valores invalidos de enum no son typos sino codificacion de informacion estructurada en texto libre. Ejemplo: `estado: "Pending (blocked by T001)"` codifica dos cosas: (1) estado bloqueado y (2) dependencia a T001. El detector migrate_value debe reconocer el patron `"ValidEnum (extra info)"`, extraer la informacion estructurada, y proponer migracion a `estado: Bloqueada` + `[[blocks:T001]]` wiki-link en body.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/proposal
interfaces:
  - nombre: ParseBlockingInfo
    metodos:
      - nombre: ParseBlockingInfo
        input: "value string"
        output: "base string, targets []string, notes []string"
  - nombre: detectMigrateValue
    metodos:
      - nombre: detectMigrateValue
        input: "enumErrors []enumError, schema *rules.StemFile"
        output: "[]Proposal"
dependencias_externas: []
tests:
  - ParseBlockingInfo("Pending (blocked by T001)") → base="Pending", targets=["T001"], notes=[]
  - ParseBlockingInfo("Pending (blocked by E04/F01)") → targets=["E04/F01"]
  - ParseBlockingInfo("Pending (blocked by E04 + E03/F05 + human)") → targets=["E04","E03/F05"], notes=["human"]
  - "Pending (blocked by T001)" con enum [Pending, Bloqueada] → migrate_value proposal
  - Valor sin parentesis → no match, pasa a correct_value
```

## Alcance

**In**:
1. Crear `internal/proposal/parse.go` con `ParseBlockingInfo()`
2. Regex para detectar patron `"BaseValue (blocked by TARGET)"` y variantes
3. Parsear compound targets: `"E04 + E03/F05 + human"` → split por ` + `, clasificar targets (patron ID-like) vs notes (texto libre como "human")
4. Agregar `detectMigrateValue()` en `detect.go` — genera propuestas con `NewValue: "Bloqueada"` y `WikiLinks: ["T001"]`
5. Integrar en `Analyze()` — migrate_value tiene prioridad 2 (despues de extend_enum)
6. Tests

**Out**: No aplicar las propuestas — eso es S003/T002. Solo detectar y generar la propuesta.

## Estado inicial esperado

- `internal/proposal/proposal.go` existe con tipos base (de T001)
- `internal/proposal/detect.go` existe con detectores basicos (de T001)
- `Analyze()` funciona para extend_enum, correct_value, add_field

## Criterios de Aceptacion

- `go test ./internal/proposal/ -run TestParseBlockingInfo -v` pasa con todos los casos
- `go test ./internal/proposal/ -run TestDetectMigrateValue -v` pasa: "Pending (blocked by T001)" → proposal tipo migrate_value
- `ParseBlockingInfo("Pending (blocked by E04 + E03/F05 + human)")` retorna targets=["E04","E03/F05"], notes=["human"]
- Valor sin parentesis no genera migrate_value (fallback a correct_value)
- `go vet ./internal/proposal/` sin errores

## Fuente de verdad

- `internal/proposal/parse.go` — archivo nuevo
- `internal/proposal/detect.go` — extender con detectMigrateValue
- `/opt/homeserver/automation/docs/epics/E03-host-restructure/` — archivos reales con "Pending (blocked by ...)" como referencia
