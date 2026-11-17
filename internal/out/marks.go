package out

import (
	"os"
	"strings"
)

var asciiMarks = strings.NewReplacer(
	"✓", "+",
	"✗", "x",
	"●", "*",
	"→", "->",
	"·", "-",
	"…", "...",
	"—", "--",
	"⚠", "!",
)

var UnicodeOK = func() bool {
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	for _, k := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := os.Getenv(k); v != "" {
			v = strings.ToLower(v)
			return strings.Contains(v, "utf-8") || strings.Contains(v, "utf8")
		}
	}
	return true
}

func Marks(s string) string {
	if UnicodeOK() {
		return s
	}
	return asciiMarks.Replace(s)
}
