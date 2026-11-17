# kongkow design principles

In short: **local-first content and destinations, one way to do each thing,
safe when in doubt, and every error says what to do next.**

Related: [command-standard.md](command-standard.md), [message-style.md](message-style.md),
[code-layout.md](code-layout.md), [extensions.md](extensions.md).

## 1. What kongkow is

kongkow is a local-first CLI for drafts/posts, collections, bookmarks and
publish destinations (Telegram, file sink). One sealed store under
`~/.kongkow` (`KONGKOW_HOME`), one vault for secrets, one safety layer. It
is for two users equally:

- **a person** at a terminal;
- **an AI agent or a script**, which gives every argument and flag and gets
  JSON back; it is never asked anything.

kongkow is **not**:

- a thin client of kongkow.xyz or any cloud content API;
- an SSH / DevOps tool (that is [komando](https://github.com/sgnd/komando)).

### Core model

```
post         = local content (title, body, media paths, collection)
destination  = where to publish (kind: telegram | file)
secret       = vault value, referenced as secret:NAME
job          = local scheduled publish
```

## 2. Safety

- Secrets never appear in argv; use `secret:NAME`.
- `--reveal` and `--yes` mean a person said yes.
- Protected destinations need `--yes` to edit, remove or publish.
- Agents prefer JSON envelopes; people can use `-o text`.

## 3. Output contract

One JSON envelope per command (`ok`, `command`, `data`, `error`, `meta`).
Fields may be added; never renamed or removed within contract 1.
Exit codes match `internal/out`.

## 4. Dual licence

AGPL-3.0 for the open-source CLI; a commercial licence is available when
the AGPL does not fit. Contributions need the CLA.
