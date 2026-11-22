// Package skills holds the agent skill that ships inside the binary.
package skills

import _ "embed"

// Kongkow is skills/kongkow/SKILL.md. `kongkow guide install` writes it.
//
//go:embed kongkow/SKILL.md
var Kongkow string
