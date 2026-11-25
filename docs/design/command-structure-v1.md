# Command structure v1 (spec)

Status: spec, not built. Target release: v0.3.0, after v0.2.0 is out.
Issue: [#19](https://github.com/sgnd/kongkow/issues/19).

This spec replaces the grammar and verb table of
[command-standard.md](command-standard.md) when it is built. Until then
that file describes the released tree. Wider rules stay in
[principles.md](principles.md); words stay in [message-style.md](message-style.md).

The structure follows the command structure of
[komando](https://github.com/sgnd/komando) (its designs v2.7 and v2.8), so
a person or an agent who knows one tool can guess the other.

## 1. Grammar

| Kind of thing | Form | Example |
|---|---|---|
| What you connect | `kongkow TYPE PROVIDER VERB NAME` | `kongkow channel telegram add tg --chat 123 --credential tg-bot` |
| kongkow's own | `kongkow NOUN VERB [NAME]` | `kongkow post publish hello --to tg` |
| Built-in and cross-type | `kongkow VERB` | `kongkow list`, `kongkow test tg`, `kongkow status` |

Rules:

- There is one type: `channel`. A provider is one network or sink.
- A provider holds only the verbs it supports.
- What you do to a thing sits under the word for that thing. The top level
  holds only built-in commands and commands that span several kinds.
- No command is deeper than four words before `NAME`.
- A name is 1 to 64 characters of `a-z`, `0-9`, `-` and `_`, starting with a
  letter or digit. Names are unique per noun; a channel name is unique across
  all providers, so `--to tg` never needs the provider.
- `@NAME` means a group wherever a name is accepted for a channel or a post.

## 2. The tree

```
kongkow
  list  test  config  update  version  schema          built-in
  channel
    telegram | mastodon | bluesky | file               what you connect
      add  list  show  test  edit  rename  remove  info
  post        create  edit  list  show  rename  remove  publish
  schedule    list  show  edit  remove  start  stop
  credential  list
    token | password    add  edit  list  show  rename  remove
  group       create  edit  list  show  rename  remove
  bookmark    add  edit  list  show  rename  remove
  status  export  import                                cross-type
  agent
    skill   install  uninstall
    mcp     start  stop
  audit       list  check  export
  doctor
```

`channel file` has no `info`: a folder has no account to describe.

Planned providers, not in v0.3.0: `threads`, `x`, `linkedin`. Each arrives
as a new provider with the same eight verbs and its own typed flags.

Not in the CLI, by design: sync, the hosted scheduler, approval from a
phone, teams, accounts and billing. They belong to the paid apps and cloud.

## 3. Verbs

The list is closed. Each verb has one meaning everywhere. The meanings are
komando's; kongkow adds only `publish`.

| Verb | The one meaning | Where |
|---|---|---|
| `add` | put a thing that exists elsewhere into kongkow | channel, credential, bookmark |
| `create` | make a new thing | post, group |
| `list` | list the things of the level you are at | top, every provider, every noun |
| `show` | everything about one thing | every provider, every noun |
| `edit` | change some fields of a thing | every provider, every noun |
| `rename` | rename a thing and every reference to it | every provider, every noun |
| `remove` | kongkow forgets a thing | every provider, every noun |
| `test` | check that something answers and lets kongkow in; change nothing | top, every provider |
| `info` | a summary read from the other side | channel providers with an account |
| `publish` | send a post to channels, now or at a time | post |
| `start` / `stop` | make something run / stop running | schedule, agent mcp |
| `status` | the state of what runs | top |
| `check` | compare with a stored value; change nothing | audit |
| `install` / `uninstall` | put the skill into an agent / take it out | agent skill |
| `get` / `set` | read / write one setting | config |
| `import` / `export` | load / save kongkow's own list | top, audit (export only) |
| `update` | install a newer kongkow | top |

Plain synonyms people type are accepted and not documented as names:
`ls` (list), `rm` (remove), `mv` (rename).

Pairs: add/remove, create/remove, start/stop, install/uninstall, get/set,
import/export. kongkow has no `delete`: nothing it removes lives on another
system. Removing a post never touches what is already live on a network.

## 4. Global flags

| Flag | Meaning |
|---|---|
| `-o json\|text` | output format; text at a terminal, json when piped |
| `--pretty` | indent JSON |
| `--dry-run` | show what would change or be sent, write and send nothing |
| `--yes` | a person confirmed an action that needs one |

`--reveal` is gone: no command shows a credential value.
There is still no `--json`, `--quiet` or TUI.

## 5. Commands

Every command prints one envelope (section 7). "Person only" is defined in
section 6. Times are RFC 3339 or `YYYY-MM-DD HH:MM` in local time.

### 5.1 Built-in

| Command | Arguments and flags | Does |
|---|---|---|
| `list` | `[--tags A,B]` | Names and one line each of every channel, post, group, bookmark and credential. Credential values never. |
| `test` | `[NAME\|@GROUP\|PATTERN ...]`, none means every channel | Runs each channel's `test`. Exit 0 only when all pass; the data lists each result. |
| `config get` | `KEY` | Prints one setting. |
| `config set` | `KEY VALUE` | Changes one setting. A safety setting is person only. |
| `config show` | | Every setting, its default, and where the files are. |
| `config vault keychain` | `--on\|--off` | Keeps the vault's key in the OS keychain instead of a file. Person only. |
| `config vault passphrase` | | Makes a new passphrase (256 bits, shown once as words) and seals everything again. The person never chooses it. Person only. |
| `update` | `[--version vX.Y.Z]` | Installs the newest release, checked against its published checksum. An older version needs `--yes`. |
| `version` | | Version, contract number and home folder. |
| `schema` | | Every command, flag, error code and exit code, machine-readable. The skill, the docs and the tree page are checked against it. |

Settings in v0.3.0:

| Key | Default | Meaning |
|---|---|---|
| `audit` | `true` | Write the audit log. Safety setting. |
| `schedule.horizon-days` | `30` | How far ahead `--next-slot` looks for a free slot. |
| `output` | unset | Default for `-o`. |

### 5.2 channel

Shared flags of `add` and `edit`, on every provider:

| Flag | Meaning |
|---|---|
| `--slots HH:MM,HH:MM` | Daily post times, local, used by `--next-slot`. Empty clears them. |
| `--protected[=false]` | Edits, removal and publishing need a person. |
| `--notes TEXT` | Free text. |
| `--tags A,B` | Tags. |

Provider flags of `add` (all also accepted by `edit`):

| Provider | Flags | Credential kind |
|---|---|---|
| `telegram` | `--chat ID` (required), `--credential NAME` (required) | `token`: the bot token |
| `mastodon` | `--server URL` (required, https), `--credential NAME` (required) | `token`: an access token |
| `bluesky` | `--credential NAME` (required), `--server URL` (default `https://bsky.social`) | `password`: the handle as user, an app password as value |
| `file` | `--dir PATH` (required) | none |

There is no `--target`, `--kind` or `--secret-ref`.

| Command | Does |
|---|---|
| `channel P add NAME <flags>` | Saves the channel. Checks the flags; does not call the network (`test` does). Fails with `EXISTS` when any channel has the name, `CREDENTIAL_NOT_FOUND` or `CREDENTIAL_KIND` when the credential is missing or of the wrong kind. |
| `channel P list` | The channels of provider P. |
| `channel P show NAME` | Every saved field. The credential is shown by name. |
| `channel P test NAME` | Calls the network's own "who am I" request; for `file`, checks the folder exists and is writable. Publishes nothing. |
| `channel P edit NAME <flags>` | Changes the given fields. Changing where posts go (`--chat`, `--server`, `--dir`) or `--credential` is person only, for every channel. |
| `channel P rename NAME NEW` | Renames it in every scheduled item and group. |
| `channel P remove NAME` | Forgets the channel. Pending scheduled items for it are cancelled and named in the output. The account is not touched. |
| `channel P info NAME` | From the network: account name, text limit, image count and size limits. Read only. |

### 5.3 post

| Command | Arguments and flags | Does |
|---|---|---|
| `post create NAME` | `--title T`, `--body TEXT` or `--body-file FILE` (`-` reads stdin), `--media A,B` (up to 4 images), `--alt TEXT` once per image in order, `--tags A,B`, `--group NAME` | Saves a new post. |
| `post edit NAME` | the same flags; an empty value clears a field | Changes the post. |
| `post list` | `--group NAME`, `--tags A,B` | Lists posts. |
| `post show NAME` | | The post, where it has gone (channel, time, link to the live post) and what is still scheduled. |
| `post rename NAME NEW` | | Renames it in every scheduled item and group. |
| `post remove NAME` | | Forgets the post. Fails with `POST_SCHEDULED` while a scheduled item is pending. Live posts on networks stay. |
| `post publish NAME` | `--to NAME,NAME\|@GROUP` (required), one of: nothing (now), `--at TIME`, `--next-slot` | See below. |

`post publish`:

- Each channel is checked before anything is sent: text length, media
  count and type, alt text. One failing channel stops the whole command and
  nothing is sent (`POST_TOO_LONG`, `MEDIA_NOT_ALLOWED`).
- Without a time flag it sends now, one channel after another. The result
  lists per channel: `ok`, the link to the live post, or the error. Exit 0
  only when every channel succeeded; a partial result exits 5 and says which
  channels are done, so a retry can name only the rest.
- `--at TIME` creates one scheduled item per channel. A time in the past is
  a usage error.
- `--next-slot` creates one scheduled item per channel at that channel's
  next free slot. `NO_SLOTS` when a channel has none; `QUEUE_FULL` when no
  slot is free inside `schedule.horizon-days`.
- `--at` and `--next-slot` exclude each other.
- `--dry-run` runs every check and prints what would be sent or scheduled.
- A `--protected` channel in `--to` makes the command person only.
- On `mastodon`, a scheduled item is handed to the server's own scheduler
  when the server supports it, so it goes out with this computer off. The
  item says which side holds it (`held_by`: `local` or `server`).

### 5.4 schedule

A scheduled item is one post going to one channel at one time. Its ID is
printed by `post publish`.

| Command | Arguments and flags | Does |
|---|---|---|
| `schedule list` | `--status pending\|done\|failed\|cancelled`, `--channel NAME`, `--post NAME` | Lists scheduled items, soonest first. |
| `schedule show ID` | | Post, channel, time, status, result and the link to the live post. |
| `schedule edit ID` | `--at TIME` or `--next-slot` | Moves a pending item. `SCHEDULE_NOT_PENDING` otherwise. |
| `schedule remove ID` | | Cancels a pending or stuck item. A done item stays in the list as history. |
| `schedule start` | `--every DURATION` (default `1m`), `--once` | Runs the scheduler in the foreground: sends what is due, then waits. `--once` does one round and exits, for cron or launchd. Only one scheduler runs per home folder (`SCHEDULER_RUNNING`). A failure writes a notice and the item becomes `failed`; it is never retried silently. |
| `schedule stop` | | Tells the running scheduler to finish its round and exit. |

Before each round the scheduler tests the credential of every channel that
has a due item, and reports a bad one without sending.

### 5.5 credential

A credential is what a channel logs in with. Its value is sealed in the
vault, is never printed, and never appears in arguments. It is referred to
by name (`--credential NAME`); in files and JSON the reference is
`credential:NAME`.

| Kind | Holds | Used by |
|---|---|---|
| `token` | one value | telegram, mastodon |
| `password` | a user name and a value | bluesky |

| Command | Arguments and flags | Does |
|---|---|---|
| `credential KIND add NAME` | `password` only: `--user NAME` (required) | Reads the value hidden at a terminal, or from stdin when piped. Person only. |
| `credential KIND edit NAME` | `password` only: `--user NAME` | Replaces the value the same way. Every channel using it follows. Person only. |
| `credential KIND list` | | Names, kind, when each changed, and the channels that use it. |
| `credential list` | | The same for every kind. |
| `credential KIND show NAME` | | The same for one. No value, no part of it. |
| `credential KIND rename NAME NEW` | | Renames it in every channel. |
| `credential KIND remove NAME` | | Forgets it. `CREDENTIAL_IN_USE` while a channel uses it. |

There is no `--value` flag and no way to read a value back. Kinds `key`
and `certificate`, `credential run` and `credential store` exist in komando
and not here; they are added when a provider needs one.

### 5.6 group

A group holds posts and channels. A group of posts is what a collection
was. A group of channels lets `--to @launch` publish to all of them.

| Command | Arguments and flags | Does |
|---|---|---|
| `group create NAME` | `--title T`, `--notes TEXT`, `--tags A,B` | Makes an empty group. |
| `group edit NAME` | `--add NAME,NAME`, `--remove NAME,NAME`, `--title`, `--notes`, `--tags` | Changes members and fields. A member is a post or a channel; when one name is both, write `post:NAME` or `channel:NAME`. |
| `group list` | | Groups with their member counts. |
| `group show NAME` | | The group and its members, by kind. |
| `group rename NAME NEW` | | Renames it everywhere. |
| `group remove NAME` | | Forgets the group. Its posts and channels stay. |

`--to @GROUP` uses only the channels of the group. A group with no channel
is a usage error there. A `--protected` channel in the group makes the
publish person only.

### 5.7 bookmark

| Command | Arguments and flags | Does |
|---|---|---|
| `bookmark add NAME` | `--url URL` (required), `--title T`, `--notes TEXT`, `--tags A,B` | Saves a link. |
| `bookmark edit NAME` | the same flags | Changes it. |
| `bookmark list` | `--tags A,B` | Lists bookmarks. |
| `bookmark show NAME` | | One bookmark. |
| `bookmark rename NAME NEW` | | Renames it. |
| `bookmark remove NAME` | | Forgets it. |

### 5.8 Cross-type

| Command | Arguments and flags | Does |
|---|---|---|
| `status` | | Whether the scheduler runs and when its last round was, the next due items, failed items not yet looked at, and whether the MCP server runs. It only shows. |
| `export` | `--file PATH` (required) | Saves channels, credentials, posts, groups, bookmarks, scheduled items and settings to one file, sealed with a generated 256-bit key that is shown once. Person only. |
| `import` | `--file PATH` (required) | Loads a file made by `export`; asks for its key hidden. Refuses when a name already exists, unless `--yes`. Person only. |

### 5.9 agent, audit, doctor

| Command | Arguments and flags | Does |
|---|---|---|
| `agent skill install` | `--agent claude-code\|codex\|cursor\|gemini-cli\|all` (default `all`), `--project` | Writes the skill for the agent. An agent may run it with `--project` or `--dry-run`; the home-wide install is person only. |
| `agent skill uninstall` | the same flags | Removes what `install` wrote, and nothing else. |
| `agent mcp start` | | Starts the local MCP server on stdio. It exposes the commands of this spec, never a credential value, and applies the same person-only rules. |
| `agent mcp stop` | | Stops a running MCP server. |
| `audit list` | `--limit N` (default 20, 0 is all), `--since TIME` | Newest entries: who, what command, on what, the result. |
| `audit check` | | Checks the audit chain was not changed. Exit 1 with the first bad entry when it was. |
| `audit export` | `--file PATH`, `--since TIME` | Saves the audit log as JSON lines. Person only. |
| `doctor` | `--offline` | Checks the setup, the store, the vault, and the credential of every channel with a pending item. `--offline` skips the network. |

The text that tells an agent how to use kongkow lives in the skill file
that `agent skill install` writes. There is no `guide` command.

## 6. Person only

An agent is a process with one of the agent variables set
(`KONGKOW_AGENT`, `CLAUDECODE`, `CODEX_THREAD_ID`, `CURSOR_AGENT`,
`GEMINI_CLI`), whatever its value, or one without a terminal. kongkow never
asks an agent anything. A person-only command run by an agent fails with
`NEEDS_PERSON` (exit 6) and a hint that names the exact command for the
person to run. `--yes` from an agent does not change that.

The table is the rule. A test walks every command path of `schema` and
fails when a path has no row here.

| Command path | Agent | Person |
|---|---|---|
| every `list`, `show`, `info`, `test`, `status`, `doctor`, `version`, `schema`, `config get`, `config show`, `audit list`, `audit check` | allowed | allowed |
| `post create`, `post edit`, `post rename`, `post remove` | allowed | allowed |
| `group *`, `bookmark *` | allowed | allowed |
| `post publish` to channels that are not protected | allowed | allowed |
| `post publish` with a protected channel | refused | `--yes` |
| `schedule edit`, `schedule remove` | allowed, unless the channel is protected | allowed; protected needs `--yes` |
| `schedule start`, `schedule stop` | allowed | allowed |
| `channel P add` with `--credential` (telegram, mastodon, bluesky) | refused | allowed |
| `channel file add` | allowed | allowed |
| `channel P edit`: `--slots`, `--notes`, `--tags` | allowed, unless the channel is protected | allowed; protected needs `--yes` |
| `channel P edit`: `--chat`, `--server`, `--dir`, `--credential`, `--protected` | refused | `--yes` |
| `channel P rename`, `channel P remove` | refused | `--yes` when protected |
| `credential KIND add`, `edit`, `rename`, `remove` | refused | allowed |
| `config set` of a safety setting, `config vault *` | refused | `--yes` |
| `config set` of any other setting | allowed | allowed |
| `export`, `import`, `audit export` | refused | allowed |
| `update` | refused | allowed |
| `agent skill install`, `uninstall` with `--project` or `--dry-run` | allowed | allowed |
| `agent skill install`, `uninstall` home-wide | refused | allowed |
| `agent mcp start`, `agent mcp stop` | allowed | allowed |

Why `channel P add` with a credential and the target flags are person only: a channel pairs
a stored credential with a place to send it. An agent that could choose the
place could send a stored token to a host of its choosing.

## 7. Output

One JSON envelope per command, as today: `ok`, `command`, `data`, `error`,
`meta`. `command` is the full path with spaces, such as
`channel telegram add`.

This release is **contract 2**: fields were renamed. Inside contract 2,
fields may be added and are never renamed or removed.

| Contract 1 | Contract 2 |
|---|---|
| `dest`, `dests` | `channel`, `channels` |
| `kind` | `provider` |
| `target` | `chat` (telegram), `server` (mastodon, bluesky), `dir` (file) |
| `secret_ref` (`secret:NAME`) | `credential` (`credential:NAME`) |
| `collection`, `collections` | `group`, `groups` |
| `jobs`, a job | `schedule`, a scheduled item |

Records in `data`:

| Record | Fields |
|---|---|
| channel | `name`, `provider`, the provider's own field (`chat`, `server` or `dir`), `credential`, `protected`, `slots`, `tags`, `notes`, `created_at`, `updated_at` |
| post | `name`, `title`, `body`, `media`, `alt`, `groups`, `tags`, `notes`, `created_at`, `updated_at`; in `show` also `published` (list of `channel`, `at`, `url`) and `scheduled` (list of item IDs) |
| scheduled item | `id`, `post`, `channel`, `at`, `status` (`pending`, `done`, `failed`, `cancelled`), `held_by` (`local`, `server`), `result`, `url`, `created_at`, `updated_at` |
| credential | `name`, `kind`, `user` (password only), `used_by`, `created_at`, `updated_at`. Never a value. |
| group | `name`, `title`, `posts`, `channels`, `tags`, `notes`, `created_at`, `updated_at` |
| bookmark | `name`, `url`, `title`, `tags`, `notes`, `created_at`, `updated_at` |

A test fails when any output holds a contract 1 field name.

Text printed back from a network (an error body, an account name) is
cleaned before it is shown or saved: no credential value, no control
characters.

## 8. Errors and exit codes

Exit codes do not change: 0 ok, 1 error, 2 usage, 3 not found, 4 connect,
5 remote, 6 policy, 7 timeout, 8 conflict.

| Code | Exit | When | Was |
|---|---|---|---|
| `CHANNEL_NOT_FOUND` | 3 | no channel has the name | `DEST_NOT_FOUND` |
| `CREDENTIAL_NOT_FOUND` | 3 | no credential has the name | `SECRET_NOT_FOUND` |
| `CREDENTIAL_KIND` | 2 | the credential is not the kind the provider needs | new |
| `CREDENTIAL_IN_USE` | 8 | removing a credential a channel uses | new |
| `GROUP_NOT_FOUND` | 3 | no group has the name | `COLLECTION_NOT_FOUND` |
| `SCHEDULE_NOT_FOUND` | 3 | no scheduled item has the ID | `JOB_NOT_FOUND` |
| `SCHEDULE_NOT_PENDING` | 8 | editing or removing an item that is done | `JOB_NOT_PENDING` |
| `SCHEDULER_RUNNING` | 8 | a second `schedule start` in one home | new |
| `POST_SCHEDULED` | 8 | removing a post with a pending item | new |
| `PROTECTED_CHANNEL` | 6 | a protected channel without a person's `--yes` | `PROTECTED_TARGET` |
| `NEEDS_PERSON` | 6 | a person-only command run by an agent | kept |
| `REVEAL_NEEDS_PERSON` | | removed with `--reveal` | removed |
| `SECRET_DESTINATION` | | removed: an agent cannot set a target any more | removed |

Unchanged: `POST_NOT_FOUND`, `BOOKMARK_NOT_FOUND`, `EXISTS`, `USAGE`,
`POST_TOO_LONG`, `MEDIA_NOT_ALLOWED`, `NO_SLOTS`, `QUEUE_FULL`,
`CONFIRM_REQUIRED`, `SAFETY_SETTING`, `STORE_NEWER`, `STORE_KEY_CHANGED`,
`STATE_CORRUPT`, `OTHER_HOME`, `WRITE_FAILED`.

Every error carries a `hint` that says what to do next. A removed command
gives a plain `USAGE` error; there is no alias and no pointer to the new
name.

## 9. Old to new

| v0.2.0 | v0.3.0 |
|---|---|
| `dest add NAME --kind K --target T --secret-ref secret:S` | `channel K add NAME <typed flags> --credential S` |
| `dest edit\|list\|show\|rm\|test` | `channel K edit\|list\|show\|remove\|test` |
| `secret set NAME [--value V]` | `credential token\|password add NAME` |
| `secret list`, `secret rm NAME` | `credential list`, `credential KIND remove NAME` |
| `secret show NAME --reveal` | removed |
| `post draft`, `post compose` | `post create` |
| `post rm` | `post remove` |
| `post schedule NAME --to D --at T` | `post publish NAME --to D --at T` |
| `post queue NAME --to D` | `post publish NAME --to D --next-slot` |
| `post status` | `schedule list`, `status` |
| `post status --run [--every 1m]` | `schedule start --once`, `schedule start --every 1m` |
| `post cancel JOB` | `schedule remove ID` |
| `collection add\|list\|show\|rm` | `group create\|list\|show\|remove` |
| `--collection NAME` | `--group NAME` |
| `bookmark rm` | `bookmark remove` |
| `guide` | the skill file |
| `guide install [--project]` | `agent skill install [--project]` |
| `audit [--limit N]` | `audit list [--limit N]` |

## 10. Store migration

The store under `KONGKOW_HOME` moves from format N to N+1 in one step, the
first time a v0.3.0 command that writes is run.

| Step | Rule |
|---|---|
| 1 | Take the single-instance lock. Refuse when a scheduler is running. |
| 2 | Copy the store and the vault to `backup-before-v0-3-0/` in the home folder, mode 600. |
| 3 | Each destination becomes a channel: `kind` to `provider`; `target` to `chat`, `server` or `dir` by provider. For bluesky the old `target` (the handle) becomes the `user` of its credential. |
| 4 | Each vault secret becomes a credential of the kind its channel needs: `token` for telegram and mastodon, `password` for bluesky. A secret no destination uses becomes a `token`. `secret:NAME` becomes `credential:NAME`. |
| 5 | Each collection becomes a group; each post's `collection` becomes membership of that group. |
| 6 | Each job becomes a scheduled item with `held_by: local`; `dest` becomes `channel`. |
| 7 | Write the new format number and one audit entry that names the counts moved. |

- The migration is all or nothing: on any error the backup is put back and
  the command fails with `STATE_CORRUPT` and a hint.
- It never prints or logs a value.
- A v0.2.0 binary that opens the new store fails with `STORE_NEWER`, as it
  does today.
- `--dry-run` on any command reports that a migration would run and what it
  would move, and changes nothing.
- The backup folder is removed by the person, never by kongkow.

## 11. Cryptography

- Vault and export use 256-bit symmetric keys (AES-256-GCM today).
- A secret a person holds is generated at 256 bits and shown once: the
  vault passphrase and the export key. kongkow refuses a chosen one.
- kongkow's network calls use Go's TLS client with the hybrid key exchange
  X25519MLKEM768 offered; the networks decide whether it is used.
- Gap, recorded: the networks' own login (bot tokens, OAuth tokens, app
  passwords) has no post-quantum option. It waits on them.

## 12. Tests that must exist

| Test | Fails when |
|---|---|
| Tree | a command exists that this spec does not list, or the other way round (read from `schema`) |
| Person-only table | a command path has no row in section 6, or an agent run of a "refused" row does anything but `NEEDS_PERSON` |
| No old names | any help text, output field, error code or doc holds `dest`, `secret`, `collection`, `guide`, `--kind`, `--target`, `--secret-ref`, `--reveal` as a command, flag or field |
| No value out | a credential value, set in a test, appears in any output, error, log, audit entry, export header or process argument |
| Publish is atomic in checks | one failing channel sends nothing to the others |
| Migration | a v0.2.0 home with each provider, a protected channel, an unused secret, a collection and jobs in every status migrates to the expected v0.3.0 home; a forced failure restores the backup byte for byte |
| Skill matches | the command list in `skills/kongkow/SKILL.md` and `AGENTS.md` equals `schema` |
| Agents can use it | ten fixed tasks (add a post, dry-run, publish to a file channel, schedule, move a scheduled item, cancel it, list what failed, read a channel's limits, find why a publish was refused, install the skill in a project) pass with two agent models and one person |

## 13. What ships in the same release

- `skills/kongkow/SKILL.md`, `AGENTS.md`, `README.md`, `docs/self-host.md`,
  `command-standard.md` and `principles.md` (core model) with the new names.
- The product site's CLI docs.
- `CHANGELOG.md`: one section "Changed, breaking" with the table of
  section 9.

## 14. Out of scope

- `post create --file` for many posts at once, and recycling a post.
- Providers `threads`, `x`, `linkedin`.
- `credential store` (secrets kept in an outside manager).
- Everything paid: sync, hosted scheduler, approval, teams.
