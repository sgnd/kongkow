# Changelog

All notable changes to kongkow. Each release has a section `## vX.Y.Z - YYYY-MM-DD`;
the release workflow takes its notes from that section.

## v0.2.0 - 2026-11-22

Update now. Earlier versions could save and print your Telegram bot token when a send failed with a network error. This version removes it from saved jobs and the audit log. If any Telegram send ever failed, rotate the bot token with BotFather, and delete older kongkow backups.

- `guide install`: teaches Claude Code (`~/.claude/skills`), Codex, Cursor and Gemini CLI (`~/.agents/skills`) to use kongkow. `--project` installs into the current folder, `--dry-run` shows the plan, and a changed file is kept as `.bak`. It writes those two files and sends nothing anywhere. It adds no guard hook: kongkow protects `~/.kongkow` itself.
- Publish proof: `post publish` and `post status` show the live link of each post (Mastodon, Bluesky, a public Telegram channel) or the file path. A private Telegram chat shows the message number. The link is stored on the job.
- `doctor` tests the token of every destination that has a scheduled post, so a dead token shows up before the post is due (`--offline` skips it), and lists jobs that failed in the last 7 days.
- A scheduled job that fails writes a `NOTICE` line to stderr when `post status --run` finds it.
- `skills/kongkow/SKILL.md`, `AGENTS.md` and the README list exactly the commands the binary has; a test fails when they drift. The skill no longer says "when releases exist".
- Release headings are `## vX.Y.Z` and the release title is the tag.

## v0.1.1 - 2026-11-18

Please update. In v0.1.0, setting `HOME` could move an AI agent's store
past the `KONGKOW_HOME` guard. v0.1.1 fixes it.

- An AI agent's kongkow now also stops when a changed `HOME` would move
  its data folder away from `~/.kongkow`. Before, only `KONGKOW_HOME` was
  checked.
- `--media` takes regular files only, so a device or folder cannot hang a
  publish.
- A file destination never replaces a post it wrote before. Two posts in
  the same second used to share one name; the second now gets `_2`.
- CI runs the built binary through its main commands (`scripts/smoke.sh`).

## v0.1.0 - 2026-11-18

- Images in `--media` are uploaded to Mastodon and Bluesky (up to 4 JPEG,
  PNG, GIF or WebP files; 1 MB each on Bluesky). Every file is checked
  before anything is sent.
- `--alt` gives each image a description for screen readers, on Mastodon
  and Bluesky.
- Links and @mentions in Bluesky posts are clickable. A mention whose
  handle cannot be found stays plain text.
- Queue slots: `dest edit NAME --slots 09:00,18:00` sets daily times, and
  `post queue NAME --to DEST` takes the next free one.
- An AI agent's kongkow stops when `KONGKOW_HOME` names another folder
  than `~/.kongkow`. Any agent variable that is set counts, even empty.

## v0.0.1 - 2026-11-18

- Publish to Mastodon (instance + access token) and Bluesky (handle + app
  password). Text limits (500 and 300 characters) are checked before
  anything is sent or scheduled.
- `post publish` and `post schedule` take `--to A,B`. Every destination
  is checked before anything is sent.
- `post status --run --every 1m` keeps running as a local scheduler;
  see docs/self-host.md for launchd and systemd.
- Local posts, collections, bookmarks and destinations in a sealed store.
- Publish to Telegram and a file sink; dry-run first.
- Secrets in a sealed vault, referenced as `secret:NAME`.
- Local scheduled publishes: `post schedule`, `post status --run`, `post cancel`.
  A due job is claimed before it is sent, so it is never sent twice.
- `doctor` warns about a job stuck running.
- `post status --run` exits 1 when a job fails.
- Sealed audit log of every change.
