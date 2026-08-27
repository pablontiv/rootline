---
tipo: adr
estado: accepted
fecha: '2026-08-27'
contexto: 'ParseStem emitia errores de texto sin tipado para sintaxis YAML y rechazos semanticos; EvaluateStemState los publicaba siempre como yaml-valid y propagaba mensajes con ruta absoluta en stem_health.'
decision: 'Introducir un error tipado interno de ParseStem con metadatos de diagnostico (check, field opcional, path y reason sin ruta) y proyectar esos metadatos en stem_health con errors.As; la sintaxis YAML permanece en yaml-valid y los rechazos semanticos de contrato (.stem nulo o version no soportada) se publican como schema-valid.'
alternativas: 'Mantener parseo por cadenas en EvaluateStemState fue descartado por fragilidad y deuda; agregar casos especiales por mensaje fue descartado por acoplar salida publica a textos internos; mover toda la clasificacion a capas de salida fue descartado porque pierde informacion estructurada en ParseErrors.'
consecuencias: 'Se estabiliza una taxonomia publica clara entre errores de sintaxis y de contrato de esquema, stem_health deja de filtrar rutas absolutas en el mensaje para rechazos semanticos, y los consumidores directos de Error() conservan contexto completo con ruta para depuracion.'
---
# 0004. Clasificar rechazos semanticos de stem como schema valid

## Contexto
ParseStem emitia errores de texto sin tipado para sintaxis YAML y rechazos semanticos; EvaluateStemState los publicaba siempre como yaml-valid y propagaba mensajes con ruta absoluta en stem_health.

## Decisión
Introducir un error tipado interno de ParseStem con metadatos de diagnostico (check, field opcional, path y reason sin ruta) y proyectar esos metadatos en stem_health con errors.As; la sintaxis YAML permanece en yaml-valid y los rechazos semanticos de contrato (.stem nulo o version no soportada) se publican como schema-valid.

## Alternativas descartadas
Mantener parseo por cadenas en EvaluateStemState fue descartado por fragilidad y deuda; agregar casos especiales por mensaje fue descartado por acoplar salida publica a textos internos; mover toda la clasificacion a capas de salida fue descartado porque pierde informacion estructurada en ParseErrors.

## Consecuencias
Se estabiliza una taxonomia publica clara entre errores de sintaxis y de contrato de esquema, stem_health deja de filtrar rutas absolutas en el mensaje para rechazos semanticos, y los consumidores directos de Error() conservan contexto completo con ruta para depuracion.
