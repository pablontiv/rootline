# Definition of Done — rootline

> Una tarea NO está terminada hasta que cumple TODOS estos criterios.

- [ ] `just check` pasa (vet + lint)
- [ ] `just test` pasa (0 failures)
- [ ] Commit convencional + push
- [ ] CLI instalado y verificado en el sistema:
  ```bash
  just install          # build local de desarrollo → ~/bin, con ldflags
  just doctor-install   # falla si hay más de un rootline en PATH
  rootline --version    # muestra versión nueva, nunca "dev"
  ```
  `doctor-install` usa `which -a`, no `which`. `which` a secas devuelve solo
  la primera coincidencia del PATH, así que un binario viejo sombreado en
  otro directorio pasaba el check sin ser visto.

  Ojo: `just install` **no** es todavía la única vía de instalación. `install.sh`
  (el instalador público) elige `~/.local/bin` cuando está en el PATH, y los
  hooks `post-merge`/`pre-push` reconstruyen sobre lo que devuelva
  `command -v rootline`. Reconciliar los tres es deuda abierta — ver
  `docs/roadmap/T011`.
