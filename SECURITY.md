# Security

kongkow holds local drafts and destination credentials (API tokens, bot
tokens) on this computer. A security bug can expose them.

## Report a problem

Please don't open a public issue for a security problem. Report it privately instead:

1. Open the [Security tab](https://github.com/sgnd/kongkow/security) of this repo.
2. Click **Report a vulnerability**.

Include what you found, how to reproduce it, and what an attacker could do with it. You get an answer within a week. Once it is fixed, a new release says what changed.

## What counts

- A way to read a secret from the vault without a human approving it, for example through JSON output (`-o json`), an error message or a log.
- A way for an AI agent to publish or send a secret somewhere without a person's yes.
- Anything that lets an AI agent do more than the person allowed.

## How kongkow keeps you safe

- Passwords and tokens live in an encrypted vault in `~/.kongkow/`. An AI agent refers to a secret as `secret:NAME` and does not see the values.
- An AI agent's kongkow uses your usual data folder: another `KONGKOW_HOME` should need a person's yes (when that guard lands).
- `secret show --reveal` works only for a person at a terminal.
- Do not read `~/.kongkow`, `store.key` or `.env` files on this computer as an agent, and do not change `KONGKOW_HOME`.

## Supported versions

Only the latest release gets security fixes. Update with the install script or from [Releases](https://github.com/sgnd/kongkow/releases).
