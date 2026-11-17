// Command kongkow manages local content drafts and publish destinations,
// for people and AI agents.
package main

import (
	"os"

	"github.com/sgnd/kongkow/internal/cli"
)

func main() { os.Exit(cli.Execute()) }
