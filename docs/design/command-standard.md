# The kongkow command standard

Grammar, names, flags and output. Wider rules: [principles.md](principles.md).
Words: [message-style.md](message-style.md).

## Grammar

`kongkow <noun> <verb> [ITEM] [args] [flags]`

Examples:

```
kongkow post draft hello --body 'Hi'
kongkow post publish hello --to local
kongkow dest add tg --kind telegram --target 123 --secret-ref secret:tg
kongkow secret set tg
```

Top-level commands include: `version`, `doctor`, `list`, `guide`, `secret`,
`dest`, `post`, `collection`, `bookmark`, `audit`.

## Verbs

| Verb | Meaning |
|---|---|
| list / show | read |
| add / draft / compose | create |
| edit | change |
| rm | remove |
| test | check a destination |
| publish | send a post now |
| schedule / status | local jobs |
| set / show / rm | vault secrets |

## Flags

- `-o json|text` — output format (default: text at a terminal, json when piped)
- `--dry-run` — show what would change / publish without doing it
- `--yes` — person confirmed a protected action
- `--reveal` — person at a terminal asked to see a secret

No `--json`, `--quiet` or TUI.

## Errors

Typed errors via `out.Errf` with stable `code`, `message`, `hint` and exit
code. Agents read `error.hint`.
