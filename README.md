# kongkow

[![CI](https://github.com/sgnd/kongkow/actions/workflows/ci.yml/badge.svg)](https://github.com/sgnd/kongkow/actions/workflows/ci.yml)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)

**All-in-one social media for the command line.** Draft once, schedule, and publish — you and your AI agent, same CLI.

kongkow is a **local-first open-source CLI** for social posts: write on this computer, keep tokens in a sealed E2EE vault (`~/.kongkow`), and ship to the destinations you connect. It is not a thin client of [kongkow.xyz](https://kongkow.xyz) or any cloud content API.

Today the OSS binary publishes to **Telegram** and a **file** sink, with local schedule and audit. More networks belong on the product roadmap; this repo stays honest about what the binary can do right now.

- **One place** for drafts, destinations, collections and bookmarks.
- **Write once, publish to the destinations you add** — dry-run first, then send.
- **Secrets stay in a sealed vault.** An AI agent never sees them.
- **Readable text for you, JSON for an agent.**
- **Local schedule and audit** for publishes and mutations.
- **One static binary**, pure Go (`CGO_ENABLED=0`).

> Tip (Bahasa): *Kongkow* = ngobrol santai. All-in-one social media di mesin Anda —
> draft, jadwal, publish — bukan klien tipis ke API cloud.

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

- Product site: [kongkow.xyz](https://kongkow.xyz) (separate from this local-first OSS CLI).
- Server and DevOps work belongs in [komando](https://github.com/sgnd/komando),
  not here.
- Future Sync/Team ideas are sketched only in
  [docs/design/extensions.md](docs/design/extensions.md) — not implemented.
