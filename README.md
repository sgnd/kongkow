# kongkow

[![CI](https://github.com/sgnd/kongkow/actions/workflows/ci.yml/badge.svg)](https://github.com/sgnd/kongkow/actions/workflows/ci.yml)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)

**Local-first posts and publish destinations. For you and your AI agent.**

kongkow keeps content and destination credentials on this computer
(`~/.kongkow`). It is **not** a thin client of a cloud API. You write
posts locally, store tokens in an encrypted vault, and publish to
destinations you choose (Telegram and a file sink).

- **One list** of posts, destinations, collections and bookmarks.
- **Secrets stay in a sealed vault.** An AI agent never sees them.
- **Readable text for you, JSON for an agent.**
- **Local schedule and audit** for publishes and mutations.
- **One static binary**, pure Go (`CGO_ENABLED=0`).

> Tip (Bahasa): *Kongkow* = ngobrol santai. CLI-nya tetap lokal di mesin
> Anda — bukan klien tipis ke API cloud.

## Install

From a clone:

```bash
make build
./bin/kongkow version
```

When releases exist:

```bash
curl -fsSL https://raw.githubusercontent.com/sgnd/kongkow/main/install.sh | sh
```

## First commands

```bash
kongkow guide
kongkow doctor
kongkow post draft hello --title Hi --body 'Hello from kongkow'
kongkow dest add local --kind file --target /tmp/kongkow-out
kongkow dest test local
kongkow post publish hello --to local
kongkow audit --limit 10
```

Telegram (token in the vault, never on the command line):

```bash
kongkow secret set tg_bot          # paste the bot token at the prompt
kongkow dest add tg --kind telegram --target CHAT_ID --secret-ref secret:tg_bot
kongkow dest test tg
kongkow post publish hello --to tg
```

## Agents

See [AGENTS.md](AGENTS.md) and `skills/kongkow/SKILL.md`. Prefer `-o json`,
never add `--yes` or `--reveal` on your own, and do not read `~/.kongkow`.

## License

AGPL-3.0 for the open-source CLI. A [commercial license](COMMERCIAL-LICENSE.md)
is available when the AGPL does not fit. See [CLA.md](CLA.md) for contributions.

## Related

Server and DevOps work belongs in [komando](https://github.com/sgnd/komando),
not here. Future Sync/Team ideas are sketched only in
[docs/design/extensions.md](docs/design/extensions.md) — not implemented.
