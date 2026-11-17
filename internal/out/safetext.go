package out

import (
	"fmt"
	"regexp"
	"strings"
)

var sgr = regexp.MustCompile(`^\x1b\[[0-9;]*m`)

func SafeText(s string) string {
	if !needsSafe(s) {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		switch {
		case r == '\x1b':
			if m := sgr.FindString(s[i:]); m != "" {
				b.WriteByte('\x1b')
				continue
			}
			b.WriteString(`\e`)
		case r == '\t' || r == '\n' || r == '\r':
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f:
			fmt.Fprintf(&b, `\x%02x`, r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func noteText(s string) string {
	return strings.NewReplacer("\x1b", `\e`, "\r", `\r`).Replace(SafeText(s))
}

func needsSafe(s string) bool {
	for _, r := range s {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f || r >= 0x80 && r <= 0x9f {
			return true
		}
	}
	return false
}
