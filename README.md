# kongkow

[![CI](https://github.com/sgnd/kongkow/actions/workflows/ci.yml/badge.svg)](https://github.com/sgnd/kongkow/actions/workflows/ci.yml)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)

**One local command line to draft, schedule and publish your social media posts. For you and your AI agent.**

You may write a post in one app, schedule it in another and keep the
tokens for each network somewhere else again. kongkow keeps all of it on
your own computer, behind one command that you and an AI agent (Claude
Code, Codex, Cursor, Gemini CLI, your own bot) can both use safely:

- **One place** for drafts, destinations, collections and bookmarks.
- **Write once, send to many.** Telegram, Mastodon, Bluesky and a file
  folder today, with images on Mastodon and Bluesky. Dry-run first, then
  publish.
- **Your own schedule.** Daily post times per destination, and a
  scheduler that runs on your machine.
- **Tokens stay in an encrypted vault** in `~/.kongkow`, sealed with a
  key on this computer. An AI agent never sees them.
- **Readable text for you, JSON for an agent.**
- **A record of everything** that was published or changed, and by whom.
- **One file to install**, no account, no telemetry.

*Kongkow* is Indonesian for a relaxed chat with friends.

## Install

On macOS or Linux, open a terminal and paste:

```bash
curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/sgnd/kongkow/main/install.sh | sh
```

On Windows, download `kongkow_windows_amd64.zip` from
[Releases](https://github.com/sgnd/kongkow/releases) and put `kongkow.exe`
in a folder on your `PATH`.

Check that it works:

```bash
kongkow version
```

To build from a clone instead: `make build`, then `./bin/kongkow version`.

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

Mastodon (an access token with `write:statuses`) and Bluesky (an app password):

```bash
kongkow secret set masto
kongkow dest add masto --kind mastodon --target https://mastodon.social --secret-ref secret:masto
kongkow secret set bsky
kongkow dest add bsky --kind bluesky --target you.bsky.social --secret-ref secret:bsky
kongkow post publish hello --to masto,bsky --dry-run
```

## Commands

| Command | What it does |
|---|---|
| `post draft / compose / edit / list / show / rm` | Write and keep posts, with images and alt text |
| `post publish` | Send a post to one or more destinations; prints the live link of each |
| `post schedule / queue / cancel / status` | Schedule a post, take the next daily slot, cancel, and list jobs (`status --run` sends the due ones and shows each live link) |
| `dest add / edit / list / show / rm / test` | Telegram, Mastodon, Bluesky and file destinations |
| `secret set / list / show / rm` | The encrypted vault for tokens |
| `collection add / list / show / rm` | Group posts |
| `bookmark add / list / show / rm` | Keep links |
| `audit` | The local record of what was published or changed |
| `doctor` | Check the store, the vault and the token of every destination with a scheduled post (`--offline` skips the token test) |
| `guide install` | Teach Claude Code, Codex, Cursor and Gemini CLI to use kongkow |
| `version` / `list` / `guide` / `completion` | About kongkow, the playbook, shell completion |

## Run the scheduler

`kongkow post status --run --every 1m` sends scheduled posts on time.
Run it under launchd or systemd on your own machine:
[docs/self-host.md](docs/self-host.md).

Every post that goes out keeps its proof: the live link (Mastodon, Bluesky,
a public Telegram channel) or the file path, shown by `post status`. A job
that fails writes a `NOTICE` line to stderr, and `kongkow doctor` lists
failures of the last 7 days and tests the token of every destination that
has a scheduled post, so a dead token shows up before the post is due.

## Privacy

No telemetry. kongkow sends nothing except the publishes you ask for.

## Agents

```bash
kongkow guide install            # Claude Code, Codex, Cursor, Gemini CLI
kongkow guide install --project  # only for the folder you are in
kongkow guide install --dry-run  # show where it would write
```

It writes the kongkow skill to `~/.claude/skills/kongkow/SKILL.md` (Claude
Code) and `~/.agents/skills/kongkow/SKILL.md` (Codex, Cursor, Gemini CLI and
others), and nothing else. It sends nothing anywhere. Unlike komando's
installer it adds no guard hook: kongkow protects `~/.kongkow` itself. Run it
again after an update: a changed file is kept as `.bak`.

See [AGENTS.md](AGENTS.md) and `skills/kongkow/SKILL.md`. Prefer `-o json`,
never add `--yes` or `--reveal` on your own, and do not read `~/.kongkow`.

## License

AGPL-3.0 for the open-source CLI. A [commercial license](COMMERCIAL-LICENSE.md)
is available when the AGPL does not fit. See [CLA.md](CLA.md) for contributions.

## Related

- Product site: [efolusi.com](https://efolusi.com) (separate from this local-first OSS CLI).
- Server and DevOps work belongs in [komando](https://github.com/sgnd/komando),
  not here.
- Future Sync/Team ideas are sketched only in
  [docs/design/extensions.md](docs/design/extensions.md). None of it is built.
