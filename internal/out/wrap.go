package out

import (
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

const maxTextWidth = 80

var StderrWidth = func() int {
	fd := int(os.Stderr.Fd())
	if !term.IsTerminal(fd) {
		return 0
	}
	w, _, err := term.GetSize(fd)
	if err != nil || w <= 0 || w > maxTextWidth {
		return maxTextWidth
	}
	return w
}

var StdoutWidth = func() int {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return 0
	}
	w, _, err := term.GetSize(fd)
	if err != nil || w <= 0 || w > maxTextWidth {
		return maxTextWidth
	}
	return w
}

// Wrap fills text into lines of at most width columns.
func Wrap(text string, labelWidth, width int) string {
	if width <= 0 {
		return text
	}
	first := width - labelWidth
	if first < 8 {
		first = 8
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	var b strings.Builder
	line, lim := "", first
	for _, w := range words {
		if line == "" {
			line = w
			continue
		}
		if utf8.RuneCountInString(line)+1+utf8.RuneCountInString(w) > lim {
			if b.Len() > 0 {
				b.WriteByte('\n')
				b.WriteString("  ")
			}
			b.WriteString(line)
			line, lim = w, width-2
			continue
		}
		line += " " + w
	}
	if b.Len() > 0 {
		b.WriteByte('\n')
		b.WriteString("  ")
	}
	b.WriteString(line)
	return b.String()
}
