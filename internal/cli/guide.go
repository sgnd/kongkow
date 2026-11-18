package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/sgnd/kongkow/internal/out"
)

const guideText = `kongkow — local-first posts and publish destinations

kongkow keeps content and destination credentials on this computer
(~/.kongkow by default). It is not a thin client of kongkow.xyz.
Server and DevOps work belongs in komando, not here.

Start
  kongkow list                 posts, destinations, collections, bookmarks
  kongkow doctor               check the store and vault
  kongkow guide                this playbook

Posts
  kongkow post draft NAME --title T --body '...' [--media a,b] [--collection C]
  kongkow post compose NAME    same as draft
  kongkow post list | show NAME | edit NAME | rm NAME
  kongkow post publish NAME --to DEST
  kongkow post schedule NAME --to DEST --at '2006-01-02 15:04'
  kongkow post cancel JOB
  kongkow post status [--run]  list jobs; --run publishes due ones

Collections and bookmarks
  kongkow collection add NAME [--title T]
  kongkow collection list | show NAME | rm NAME
  kongkow bookmark add NAME --url URL [--title T]
  kongkow bookmark list | show NAME | rm NAME

Destinations
  kongkow dest add NAME --kind file --target DIR
  kongkow dest add NAME --kind telegram --target CHAT_ID --secret-ref secret:TOKEN
  kongkow dest list | show NAME | edit NAME | rm NAME | test NAME
  Never put bot tokens on the command line: store them with secret set first.
  A --protected destination needs --yes to edit, remove or publish.

Secrets
  kongkow secret set NAME
  kongkow secret list
  kongkow secret show NAME --reveal   person at a terminal only
  kongkow secret rm NAME

Audit
  kongkow audit [--limit N]    local sealed audit log of mutations

Agents
  Prefer -o json when branching on errors; -o text to read prose.
  Never add --yes or --reveal on your own. Exit 6 means stop and ask.
  Do not read ~/.kongkow, store.key or .env, and do not change KONGKOW_HOME.
  Dry-run publishes with: kongkow post publish NAME --to DEST --dry-run

Future (not in this open-source CLI)
  Paid sync and team features may appear later as optional extensions.
  This binary stays local-first; there is no kongkow.xyz API client here.

License
  AGPL-3.0; a commercial license is also available (COMMERCIAL-LICENSE.md).
`

type guideResult struct {
	TextBody string `json:"text"`
}

func (g guideResult) Text() string { return g.TextBody }

func guideCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "guide",
		Short: "Print the kongkow playbook for people and agents",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			body := guideText
			if len(args) == 1 {
				topic := strings.ToLower(args[0])
				if !strings.Contains(strings.ToLower(guideText), topic) {
					return out.Usage("unknown guide topic %q; try `kongkow guide`", args[0])
				}
			}
			out.P.Print("guide", guideResult{TextBody: body}, nil)
			return nil
		},
	}
}
