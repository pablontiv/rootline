---
tipo: adr
estado: accepted
fecha: "2026-08-24"
contexto: "Las reparaciones de representaciones YAML deben convertir escalares nativos a string sin perder el texto original ni debilitar los guards de reportes existentes."
decision: "Conservar lexema y representación escalar como metadatos internos del Record y discriminar las reparaciones correct_value mediante from_representation."
consecuencias: "Timestamp, boolean e integer podrán citarse de forma automática y exacta; los demás errores de tipo se informarán sin coerción y los contratos JSON existentes seguirán compatibles."
---

# ADR 0002: Preservar lexemas en reparaciones de tipo

## Contexto

Los contratos estrictos de campos rechazan correctamente una fecha, un booleano o un entero YAML nativo cuando el esquema exige `type: string`. El motor de propuestas no procesa `rule: type`, por lo que `validate` puede fallar mientras `fix` informa cero propuestas. El defecto bloquea la migración de corpus existentes que dependían de representaciones aceptadas antes del contrato estricto.

Una reparación segura necesita distinguir el valor textual que el autor escribió de la representación que YAML resolvió. Después de decodificar, `2026-06-22` y `2026-06-22T00:00:00Z` son valores `time.Time`; `042` y otras formas enteras también pierden su forma léxica. Reconstruir texto desde el valor Go puede cambiar el contenido que pasará a ser contractual cuando se cite como string.

Los reportes de reparación son entrada no confiable. El guard existente compara el valor actual con `Proposal.From` mediante igualdad estricta. Una representación nativa no puede compararse con el string serializado en JSON sin evidencia adicional, pero aplicar una comparación type-aware a todas las propuestas `correct_value` volvería permisivos los reportes antiguos.

## Decisión

`extract.Record` conservará, fuera de su JSON público, el lexema exacto y la representación canónica de escalares YAML de frontmatter. La evidencia se obtendrá del mismo `yaml.Node` usado para decodificar el mapa de frontmatter. Las representaciones reparables serán exclusivamente `timestamp`, `boolean` e `integer` cuando la validación del registro indique que el campo esperaba `string`.

`ValidationError` transportará expected y actual representation mediante campos internos `json:"-"`. El detector no analizará mensajes humanos ni consultará un stem global que pueda diferir del esquema efectivo del registro.

Las reparaciones reutilizarán `correct_value`. `Proposal` incorporará `from_representation,omitempty`; `From` y `To` contendrán el lexema real, sin sentinelas ni comillas preinsertadas. La escritura YAML convertirá el nodo a `!!str` y elegirá su estilo citado.

`repair apply` activará la comparación de lexema y representación únicamente cuando `from_representation` esté presente. Un marcador desconocido, evidencia ausente o cualquier diferencia rechazará la propuesta. Los reportes anteriores, que omiten el campo, conservarán la igualdad estricta existente.

Los errores `rule: type` fuera de la allowlist se publicarán como `type_findings` no reparables. Los hallazgos no cambiarán el exit code de `fix`; `validate` continuará siendo la autoridad sobre validez del corpus.

## Alternativas descartadas

### Reconstruir el texto desde el valor Go

Es simple, pero pierde diferencias léxicas como timestamp completo frente a fecha, mayúsculas booleanas o ceros y signos enteros. Al convertir a string, esas diferencias pasan a formar parte del dato y no pueden descartarse.

### Crear un tipo `normalize_representation`

Expresaría la operación de forma explícita, pero fragmentaría la taxonomía de propuestas, los contadores, `Surface()` y ambos aplicadores sin aportar una nueva superficie de mutación. `correct_value` más un discriminador representa la misma operación de reparación.

### Usar sentinelas como `<timestamp>` en `From`

Convierte un campo documentado como valor esperado en un protocolo de magic strings, puede colisionar con datos legítimos y no verifica cuál timestamp estaba presente al generar el reporte.

### Aplicar comparación type-aware a todos los `correct_value`

Debilita los guards de reportes existentes y permite aceptar estados que no coinciden exactamente con `From`. La rama especializada debe estar aislada por un campo explícito y opcional.

### Releer archivos desde `proposal.Analyze`

Acopla el motor de propuestas con paths e I/O, duplica extracción y puede analizar bytes distintos de los que validación procesó. La evidencia sintáctica pertenece al límite de extracción.

### Añadir evidencia al JSON público de `ValidationError`

Transportaría expected y actual representation, pero ampliaría el contrato versionado de `validate` con datos necesarios solo para coordinación interna. Los campos `json:"-"` conservan evidencia sin cambiar el envelope.

### Fallar `fix` ante hallazgos no reparables

Confundiría éxito de la operación de reparación con validez del corpus y divergiría del precedente de `link_findings`. `fix` informa; `validate` dicta el veredicto de validez.

## Consecuencias

- `fix --all` podrá reparar automáticamente fechas, timestamps, booleanos y enteros YAML cuando el contrato exija string.
- El texto original se preservará exactamente y la segunda ejecución será un no-op.
- `repair apply` rechazará reportes obsoletos si cambió el lexema o la representación, aunque el valor parseado parezca equivalente.
- Los reportes existentes conservarán su comportamiento porque `from_representation` es opcional.
- El JSON público de records y errores de validación no cambiará.
- Los envelopes de fix incorporarán campos aditivos `type_findings,omitempty` sin cambio de versión.
- Mapping, sequence, null, number y conversiones inversas permanecerán fuera de la reparación automática y serán visibles como hallazgos.
- Extracción mantendrá una pequeña cantidad adicional de metadatos por escalar de frontmatter.
- Las pruebas deberán defender preservación léxica, aislamiento del guard, compatibilidad de reportes, idempotencia y paridad entre dry-run y ejecución.