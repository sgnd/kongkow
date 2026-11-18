# Planned work

The one list of planned work for the kongkow CLI. Mark an item `done` or
`deferred` (with the reason and what it waits on). Never delete an item.

## Milestone 1: safe scheduled jobs

| Item | Status |
|---|---|
| Claim a due job (`running`) before publishing, so two runs never send it twice | done |
| `kongkow post cancel JOB` for pending jobs | done |
| CLI tests: schedule, run due jobs, cancel, failed job | done |

## Milestone 2: test coverage of the core

| Item | Status |
|---|---|
| Tests for `store` (load, update, sealed backend) | open |
| Tests for `vault` (set, resolve, hidden output) | open |
| Tests for `sealdb` audit log | open |
| Tests for `dest` Telegram with a fake HTTP server | open |
| CLI tests for collection, bookmark, secret, doctor, list, audit | open |

## Milestone 3: commands and polish

| Item | Status |
|---|---|
| Stale `running` job recovery in `doctor` (job left by a crashed run) | open |
| `post status --run` exit code when a job fails | open |
| Release workflow (tagged binaries for `install.sh`) | open, needs owner yes to tag |

## Paid plan (after the CLI is solid)

| Item | Status |
|---|---|
| `efolusi/kongkow-apps`: Mac, iPhone, iPad app, built like komando-apps | deferred: waits on milestones 1-3 |
| `efolusi/kongkow-cloud`: API, account, sync, billing, website, built like komando-cloud | deferred: waits on milestones 1-3 |
| More networks (X, Mastodon, Bluesky, LinkedIn) | deferred: product decision, after paid plan shape is set |
