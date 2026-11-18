# Code layout

| Rule | Notes |
|---|---|
| Pure Go | `CGO_ENABLED=0`; modernc.org/sqlite |
| Built in | Features work with the kongkow binary alone |
| Open source deps | MIT/BSD/ISC/Apache |
| cmd/kongkow | only starts the program |
| internal/cli | commands, flags, audit, print |
| internal/store | sealed inventory under KONGKOW_HOME |
| internal/sealdb / sealrow | sealed SQLite rows |
| internal/vault | secrets, encrypted with a key on this computer |
| internal/dest | Telegram + file publish |
| internal/out | JSON envelope + text |

Packages are one short lower-case word. Port patterns from komando; do not
import komando as a module dependency.

Functions stay small; prefer split files over growing past ~800 lines.
