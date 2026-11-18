# Changelog

All notable changes to kongkow. Each release has a section `## [X.Y.Z]`;
the release workflow takes its notes from that section.

## [0.1.0]

- Local posts, collections, bookmarks and destinations in a sealed store.
- Publish to Telegram and a file sink; dry-run first.
- Secrets in a sealed vault, referenced as `secret:NAME`.
- Local scheduled publishes: `post schedule`, `post status --run`, `post cancel`.
  A due job is claimed before it is sent, so it is never sent twice.
- `doctor` warns about a job stuck running.
- `post status --run` exits 1 when a job fails.
- Sealed audit log of every change.
