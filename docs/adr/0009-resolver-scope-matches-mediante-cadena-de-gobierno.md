---
tipo: adr
estado: superseded
fecha: '2026-08-29'
contexto: 'Stem health evaluaba scope.match solo contra archivos inmediatos, aunque el descubrimiento recursivo gobierna descendientes y --strict fallaba sobre corpus válidos.'
decision: 'Derivar para cada archivo inventariado el stem propietario del scope.match efectivo mediante Chain; la herencia alcanza descendientes y los overrides o root:true anidados conservan propiedad exclusiva.'
alternativas: 'Se descartó buscar por prefijo recursivo porque atravesaría overrides y raíces anidadas; se descartó pasar records descubiertos porque acoplaría el evaluador puro a Scan y ampliaría innecesariamente el alcance hacia el filesystem inyectable.'
consecuencias: 'Los diagnósticos coinciden con la cadena real de gobierno; el cálculo recorre una cadena por archivo y no añade estado, flags ni otra autoridad de resolución.'
superseded_by: 0010-distinguir-inventario-fisico-y-elegibilidad-de-scope
---
# 0009. Resolver scope matches mediante cadena de gobierno

## Contexto
Stem health evaluaba scope.match solo contra archivos inmediatos, aunque el descubrimiento recursivo gobierna descendientes y --strict fallaba sobre corpus válidos.

## Decisión
Derivar para cada archivo inventariado el stem propietario del scope.match efectivo mediante Chain; la herencia alcanza descendientes y los overrides o root:true anidados conservan propiedad exclusiva.

## Alternativas descartadas
Se descartó buscar por prefijo recursivo porque atravesaría overrides y raíces anidadas; se descartó pasar records descubiertos porque acoplaría el evaluador puro a Scan y ampliaría innecesariamente el alcance hacia el filesystem inyectable.

## Consecuencias
Los diagnósticos coinciden con la cadena real de gobierno; el cálculo recorre una cadena por archivo y no añade estado, flags ni otra autoridad de resolución.
