---
tipo: adr
estado: accepted
fecha: '2026-08-29'
contexto: 'La instalación procesa Claude antes que Agents y una preimagen Agents puede enlazar a Claude, por lo que el digest desreferenciado cambia debido a una acción anterior de la misma operación.'
decision: 'Verificar una preimagen staged de symlink por su tipo, destino, kind y target lexical exacto contra el plan y el backup; mantener la verificación completa de digest para directorios.'
alternativas: 'Reordenar destinos se descartó porque acopla la seguridad a un orden particular; conservar el digest desreferenciado se descartó porque la operación lo invalida por sí misma.'
consecuencias: 'La aprobación sigue vinculando el digest observado antes de ejecutar, mientras la publicación valida el objeto restaurable real y permite converger cuando un destino soportado enlaza a otro.'
---
# 0008. Verificar preimagenes symlink por identidad del enlace

## Contexto
La instalación procesa Claude antes que Agents y una preimagen Agents puede enlazar a Claude, por lo que el digest desreferenciado cambia debido a una acción anterior de la misma operación.

## Decisión
Verificar una preimagen staged de symlink por su tipo, destino, kind y target lexical exacto contra el plan y el backup; mantener la verificación completa de digest para directorios.

## Alternativas descartadas
Reordenar destinos se descartó porque acopla la seguridad a un orden particular; conservar el digest desreferenciado se descartó porque la operación lo invalida por sí misma.

## Consecuencias
La aprobación sigue vinculando el digest observado antes de ejecutar, mientras la publicación valida el objeto restaurable real y permite converger cuando un destino soportado enlaza a otro.
