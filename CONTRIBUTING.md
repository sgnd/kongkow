# Contributing

Thank you for helping. Please write issues, pull requests and docs in English.

kongkow is open source under the [AGPL-3.0](LICENSE), and is also sold under a
[commercial license](COMMERCIAL-LICENSE.md) that pays for its work. Before your
first pull request is merged, please sign the
[Contributor License Agreement](CLA.md): type its
sentence in the pull request description (every commit author signs), and a
check looks for it. You keep the copyright of what you wrote.

kongkow is a local-first CLI for drafts and publish destinations, for people and
their AI agents. It stays simple on purpose: one static binary, no menus or
windows, safe by default.

## Report a bug or suggest a feature

Open a GitHub issue. For a security problem, see [SECURITY.md](SECURITY.md) instead.

## Before you add something

A change fits when all of these are true:

- It is about local content, publish destinations or the local vault, not SSH,
  servers or DevOps (that is [komando](https://github.com/sgnd/komando)).
- It works as a command, for a person and for an AI agent.
- It is safe by default: it only reads unless the user says otherwise.
- It needs no new dependency, or the dependency is small and well kept.
- It does not talk to efolusi.com unless the feature is an explicit, optional
  extension point documented in [docs/design](docs/design).

For a bigger feature, open an issue first. Every feature follows
[docs/design/principles.md](docs/design/principles.md) and the
[command standard](docs/design/command-standard.md).

## Build and test

You need [Go](https://go.dev/dl/) (the version in [`go.mod`](go.mod) or newer),
`make` and `git`.

```bash
make build    # compile ./bin/kongkow
make test     # unit tests
make lint     # gofmt + go vet + staticcheck
```

`make help` lists everything.

## Add or change a command

Commands live in `internal/cli/`. Print results only via `out.P.Print`. Never
prompt an agent: missing input is an error with a `hint`. Typed errors use
`out.Errf`. Messages follow [docs/design/message-style.md](docs/design/message-style.md).
Where code goes is in [docs/design/code-layout.md](docs/design/code-layout.md).

## Merging outside pull requests

For maintainers. Squash-merge a pull request from outside, and add one
trailer per signer to the merge commit, so git keeps the CLA record:

```
CLA-Signed-By: <login> (#<number>, <YYYY-MM-DD>)
```

Never delete the `cla-signatures` branch: it holds the signatures given to
the old bot before 2026-11-23.
