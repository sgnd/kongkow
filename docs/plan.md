# Planned work

The one list of planned work for the kongkow CLI. Mark an item `done` or
`deferred` (with the reason and what it waits on). Never delete an item.

## Goal

kongkow is the premium social media scheduler from the internet (the job
of Buffer or Hootsuite), running locally, for people and AI agents. Like
komando: an open-source CLI first, then paid apps and a paid cloud.

| Value | What it means for kongkow |
|---|---|
| Open source, for transparency | The CLI is AGPL; every publish path is readable code. |
| Self-hosted, for ownership | Posts, jobs and the vault live in `~/.kongkow`; the scheduler runs on your own machine or server. |
| Private, for protection | No telemetry; nothing leaves the machine except the publish you asked for. |
| Secure, for trust | Tokens in the sealed vault; agents never see them; protected destinations need `--yes`. |
| Cloud, for convenience | Paid kongkow-cloud: sync, hosted scheduler, team approval. Optional, end-to-end encrypted. |
| Commercial, for sustainability | Paid apps and cloud; commercial licence for the CLI. |

## Milestone 1: safe scheduled jobs

| Item | Status |
|---|---|
| Claim a due job (`running`) before publishing, so two runs never send it twice | done |
| `kongkow post cancel JOB` for pending jobs | done |
| CLI tests: schedule, run due jobs, cancel, failed job | done |
| CI green: remove unused `loadState` (staticcheck), toolchain go1.27.4 (GO-2026-6859 in net/http) | done |

## Milestone 2: test coverage of the core

| Item | Status |
|---|---|
| Tests for `store` (load, update, sealed backend) | done |
| Tests for `vault` (set, resolve, hidden output) | done |
| Tests for `sealdb` audit log | done |
| Tests for `dest` Telegram with a fake HTTP server | done |
| CLI tests for collection, bookmark, secret, doctor, list, audit | done |

## Milestone 3: commands and polish

| Item | Status |
|---|---|
| Stale `running` job recovery in `doctor` (job left by a crashed run) | done: doctor warns after 10 min; `post cancel` clears a stuck job |
| `post status --run` exit code when a job fails | done: exit 1 |
| Release workflow (tagged binaries for `install.sh`) | done: `release.yml`, `make dist`, CHANGELOG.md |
| First release tag `v0.0.1` (semver from 0.0.1; tags group changes) | done: released |
| Release `v0.1.0` (queue slots, KONGKOW_HOME guard, Bluesky facets, images and alt text) | open, needs owner yes to tag |
| Signed macOS binaries and signed checksums (like komando) | deferred: needs Apple Developer ID and a release signing key from the owner |

## Milestone 4: the killer feature, locally

| Item | Status |
|---|---|
| Publish to Mastodon and Bluesky (open APIs first) | done (text only; media upload is a separate item) |
| Publish to X and LinkedIn | split: see the two rows below |
| Publish to LinkedIn | waiting: the owner creates the LinkedIn developer app (owner, 2026-11-18: "Ya, saya buat") |
| Publish to X | deferred: the X API costs money; waits on the owner (owner, 2026-11-18: "Tidak sekarang") |
| One post to many destinations: `post publish NAME --to A,B` | done: also `post schedule`; all checked before any is sent |
| Per-network text limits and checks before publish | done for Mastodon (500) and Bluesky (300) |
| Media upload to Mastodon and Bluesky (today media paths are listed as text) | done: up to 4 images, checked before anything is sent |
| Alt text for uploaded images (`post draft --alt`) | done: one `--alt` per image; new `--media` clears old alt texts |
| Bluesky links and mentions as rich-text facets | done: a mention whose handle does not resolve stays plain text |
| `kongkow schedule run --watch`: a local scheduler loop, plus a launchd/systemd example | done as `post status --run --every 1m` (one command for running jobs, not a second one) |
| Queue slots (post times per destination, `post queue`) | done: `dest add/edit --slots`, `post queue NAME --to A,B` |
| Self-host guide: run the scheduler on your own server | done: docs/self-host.md |
| No-telemetry statement in README and SECURITY.md | done |
| Guard: an agent's kongkow refuses another `KONGKOW_HOME` without a person's yes (SECURITY.md says "when that guard lands") | done: final stop, exit 6, no flag |

## Paid plan (after the CLI is solid)

| Item | Status |
|---|---|
| `efolusi/kongkow-apps`: Mac, iPhone, iPad app, built like komando-apps | deferred: waits on milestones 1-3 |
| `efolusi/kongkow-cloud`: API, sync, hosted scheduler, website, built like komando-cloud | deferred: waits on milestones 1-3 |
| Login and accounts | via sso: comes from `efolusi/sso`, never built in kongkow |
| Plans, billing and checkout | via pay: comes from `efolusi/pay`, never built in kongkow |
| Cloud sync end-to-end encrypted, hosted scheduler, team approval | deferred: part of kongkow-cloud |
| More networks (X, Mastodon, Bluesky, LinkedIn) | moved up: see milestone 4 (the killer feature needs them) |
