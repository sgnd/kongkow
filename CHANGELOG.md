# Changelog

All notable changes to kongkow. Each release has a section `## [X.Y.Z]`;
the release workflow takes its notes from that section.

## [0.0.1]

- Publish to Mastodon (instance + access token) and Bluesky (handle + app
  password). Text limits (500 and 300 characters) are checked before
  anything is sent or scheduled.
- `post publish` and `post schedule` take `--to A,B`. Every destination
  is checked before anything is sent.
- `post status --run --every 1m` keeps running as a local scheduler;
  see docs/self-host.md for launchd and systemd.
- Queue slots: `dest edit NAME --slots 09:00,18:00` sets daily times, and
  `post queue NAME --to DEST` takes the next free one.
- Local posts, collections, bookmarks and destinations in a sealed store.
- Publish to Telegram and a file sink; dry-run first.
- Secrets in a sealed vault, referenced as `secret:NAME`.
- Local scheduled publishes: `post schedule`, `post status --run`, `post cancel`.
  A due job is claimed before it is sent, so it is never sent twice.
- `doctor` warns about a job stuck running.
- `post status --run` exits 1 when a job fails.
- Sealed audit log of every change.
