# Extension points (future Sync / Team)

These slots are documented only. **This open-source CLI does not implement
cloud sync, team sharing, SSO or pricing.**

## Intent

Local-first remains the default. Optional paid capabilities may later plug
in without turning kongkow into a required online client.

## Suggested interfaces (not implemented)

1. **SyncProvider** — optional backup/sync of the sealed store between
   devices the person owns. Would sit behind an interface in
   `internal/sync` (package does not exist yet). The CLI would call it only
   when configured; offline use stays complete without it.

2. **TeamDirectory** — optional shared destination or collection names for
   a team. Would never replace the local vault; secrets stay local unless a
   person explicitly opts into a future shared secret store.

3. **Identity / SSO** — out of scope for the OSS binary. Any future login
   (including third-party IdPs) belongs in a separate commercial surface,
   not as a hard dependency of `kongkow` commands.

## Rules for contributors

- Do not add HTTP clients to efolusi.com, sync protocols or SSO flows in
  this repository unless a design issue explicitly opens that phase.
- Comments and empty interfaces that name Sync/Team are fine; working
  cloud code is not.
- Commercial licensing is described in COMMERCIAL-LICENSE.md; do not design
  pricing tiers or app SKUs here.
