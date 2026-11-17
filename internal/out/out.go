// Package out renders every command result as a stable, machine-readable
// envelope. AI agents parse stdout; humans can ask for -o text.
package out

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	ExitOK       = 0
	ExitError    = 1
	ExitUsage    = 2
	ExitNotFound = 3
	ExitConnect  = 4
	ExitRemote   = 5
	ExitPolicy   = 6
	ExitTimeout  = 7
	ExitConflict = 8
)

// Error is a typed error with a stable code, exit code and hint.
type Error struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Hint       string `json:"hint,omitempty"`
	Thing      string `json:"thing,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
	Exit       int    `json:"-"`
	Final      bool   `json:"-"`

	say, sayHint string
	cancelled    bool
	need         []string
}

func (e *Error) Say(msg, hint string) *Error { e.say, e.sayHint = msg, hint; return e }
func (e *Error) PlainHint() *Error           { e.sayHint = ""; return e }
func (e *Error) Cancel(msg string) *Error    { e.say, e.cancelled = msg, true; return e }
func (e *Error) TextMessage() string {
	if e.say != "" {
		return e.say
	}
	return e.Message
}
func (e *Error) TextHint() string {
	if e.sayHint != "" {
		return e.sayHint
	}
	return e.hintOrDefault()
}
func (e *Error) Error() string               { return e.Code + ": " + e.Message }
func (e *Error) NoFlag() *Error              { e.Final = true; return e }
func (e *Error) WithHint(h string) *Error    { e.Hint = h; return e }
func (e *Error) About(thing string) *Error   { e.Thing = thing; return e }
func (e *Error) Need(flags ...string) *Error { e.need = append(e.need, flags...); return e }

func (e Error) MarshalJSON() ([]byte, error) {
	type plain Error
	p := plain(e)
	p.Hint = e.hintOrDefault()
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(p); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n")), nil
}

func (e Error) hintOrDefault() string {
	if e.Hint != "" {
		return e.Hint
	}
	return DefaultHint(e.Exit)
}

func DefaultHint(exit int) string {
	switch exit {
	case ExitUsage:
		return UsageHint
	case ExitNotFound:
		return "check the name; `kongkow list` shows drafts and destinations"
	case ExitPolicy:
		return "stop and tell the user what you want to do and why; `kongkow guide` explains the safety stops"
	case ExitTimeout:
		return "give it more time with --timeout, or try again later"
	case ExitConflict:
		return "the state is not what the command expected: look at it again, then run it again"
	}
	return "read the message; `kongkow doctor` checks this computer's setup"
}

const UsageHint = "run with --help for the command reference"

func Errf(exit int, code, hint, format string, a ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, a...), Hint: hint, Exit: exit}
}

func Usage(format string, a ...any) *Error {
	return Errf(ExitUsage, "USAGE", UsageHint, format, a...)
}

type Envelope struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Data    any    `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
	Meta    Meta   `json:"meta"`
}

const Contract = 1

type Meta struct {
	Contract       int      `json:"contract"`
	Version        string   `json:"version"`
	DurationMS     int64    `json:"duration_ms"`
	DryRun         bool     `json:"dry_run,omitempty"`
	WouldBeBlocked *Error   `json:"would_be_blocked,omitempty"`
	WouldNeed      []string `json:"would_need,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
	Truncated      bool     `json:"truncated,omitempty"`
}

type Printer struct {
	Format  string
	Pretty  bool
	Version string
	Start   time.Time
	DryRun  bool
	W       io.Writer
	blocked *Error
	need    []string
	warns   []string
	mu      sync.Mutex
	cut     bool
	cutAt   int
	cutNote bool

	Secrets  func() map[string]string
	NoRedact bool
	Mask     bool
	once     sync.Once
	hmu      sync.RWMutex
	hide     []hidden
}

type hidden struct {
	raw, json, label string
	added            bool
}

const minSecret = 6

func (p *Printer) load() {
	p.once.Do(func() {
		if p.Secrets == nil {
			return
		}
		p.hmu.Lock()
		defer p.hmu.Unlock()
		for name, v := range p.Secrets() {
			p.addLocked(v, "[secret:"+name+"]", false)
		}
	})
}

func (p *Printer) addLocked(v, label string, added bool) {
	if len(v) < minSecret {
		return
	}
	p.hide = append(p.hide, hidden{raw: v, json: string(mustJSON(v)), label: label, added: added})
}

func mustJSON(s string) []byte {
	b, _ := json.Marshal(s)
	return b
}

func (p *Printer) hideList() []hidden {
	p.load()
	p.hmu.RLock()
	defer p.hmu.RUnlock()
	return slices.Clone(p.hide)
}

func (p *Printer) forgetAdded() {
	p.hmu.Lock()
	defer p.hmu.Unlock()
	kept := p.hide[:0]
	for _, h := range p.hide {
		if !h.added {
			kept = append(kept, h)
		}
	}
	p.hide = kept
}

func (p *Printer) Redact(s string) string {
	if p.NoRedact {
		return s
	}
	for _, h := range p.hideList() {
		s = strings.ReplaceAll(s, h.raw, h.label)
	}
	return s
}

func (p *Printer) redactJSON(b []byte) []byte {
	if p.NoRedact {
		return b
	}
	for _, h := range p.hideList() {
		b = bytes.ReplaceAll(b, []byte(h.json), []byte(h.label))
	}
	return b
}

var P = &Printer{Format: "json", W: os.Stdout, Start: time.Now()}

type Texter interface{ Text() string }

func (p *Printer) meta() Meta {
	p.mu.Lock()
	warns, cut := slices.Clone(p.warns), p.cut
	p.mu.Unlock()
	return Meta{
		Contract: Contract, Version: p.Version, DurationMS: time.Since(p.Start).Milliseconds(),
		DryRun: p.DryRun, WouldBeBlocked: p.blocked, WouldNeed: slices.Clone(p.need),
		Warnings: warns, Truncated: cut,
	}
}

func (p *Printer) WouldBlock(e *Error) {
	if p.blocked == nil {
		p.blocked = e
	}
	for _, f := range e.Flags() {
		if !slices.Contains(p.need, f) {
			p.need = append(p.need, f)
		}
	}
	slices.Sort(p.need)
}

func ApprovalFlag(e *Error) string {
	if e == nil || e.Final || e.Exit != ExitPolicy {
		return ""
	}
	switch e.Code {
	case "POLICY_BLOCKED":
		return "--allow-dangerous"
	case "PROTECTED_TARGET", "SAFETY_SETTING", "SECRET_DESTINATION", "CONFIRM_REQUIRED":
		return "--yes"
	case "REVEAL_NEEDS_PERSON":
		return "--reveal"
	}
	return ""
}

func (e *Error) Flags() []string {
	f := ApprovalFlag(e)
	if f == "" {
		return nil
	}
	res := []string{f}
	for _, n := range e.need {
		if !slices.Contains(res, n) {
			res = append(res, n)
		}
	}
	slices.Sort(res)
	return res
}

func (p *Printer) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blocked, p.need, p.warns = nil, nil, nil
	p.cut, p.cutNote, p.cutAt = false, false, 0
	p.forgetAdded()
}

func (p *Printer) Warn(msg string) {
	p.mu.Lock()
	p.warns = append(p.warns, msg)
	p.mu.Unlock()
	fmt.Fprintln(os.Stderr, Marks("Warning: "+Wrap(noteText(p.Redact(msg)), len("Warning: "), StderrWidth())))
}

func (p *Printer) Cut(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cut = true
	if n > p.cutAt {
		p.cutAt = n
	}
}

func (p *Printer) printCut() {
	p.mu.Lock()
	cut, noted, n := p.cut, p.cutNote, p.cutAt
	p.mu.Unlock()
	if cut && !noted {
		fmt.Fprintf(os.Stderr, "Stopped at --limit %d; more items are there (--limit 0 shows them all).\n", n)
	}
}

func (p *Printer) Print(command string, data any, softErr *Error) {
	env := Envelope{OK: softErr == nil, Command: command, Data: data, Error: softErr, Meta: p.meta()}
	if p.Format == "text" {
		if p.DryRun {
			fmt.Fprintln(os.Stderr, "Dry run: nothing was changed")
			if p.blocked != nil {
				fmt.Fprintf(os.Stderr, "Without --dry-run this stops here: [%s] %s\n", p.blocked.Code, Marks(noteText(p.Redact(p.blocked.Message))))
				p.hint(p.blocked.hintOrDefault())
			}
		}
		if t, ok := data.(Texter); ok {
			s := SafeText(p.Redact(t.Text()))
			fmt.Fprint(p.W, s)
			if s != "" && !strings.HasSuffix(s, "\n") {
				fmt.Fprintln(p.W)
			}
		} else if data != nil {
			s := renderText(data)
			fmt.Fprint(p.W, SafeText(p.Redact(s)))
			if s == "" {
				fmt.Fprintln(os.Stderr, "Nothing to show")
			}
		}
		p.printCut()
		if softErr != nil {
			p.printErr(softErr, false)
		}
		return
	}
	p.writeJSON(env)
}

func (p *Printer) Fail(command string, err error) int {
	var e *Error
	if !errors.As(err, &e) {
		e = &Error{Code: "ERROR", Message: err.Error(), Exit: ExitError}
	}
	if p.Format == "text" {
		p.printErr(e, true)
		return e.Exit
	}
	p.writeJSON(Envelope{OK: false, Command: command, Error: e, Meta: p.meta()})
	return e.Exit
}

func (p *Printer) printErr(e *Error, stopped bool) {
	msg := Marks(noteText(p.Redact(e.TextMessage())))
	if e.cancelled {
		fmt.Fprintln(os.Stderr, msg)
		return
	}
	label := "Error: "
	if strings.HasPrefix(msg, "Stopped") {
		label = ""
	} else if stopped && e.Exit == ExitPolicy {
		label = "Stopped: "
		msg = strings.TrimRight(msg, " .")
		if !strings.Contains(strings.ToLower(msg), "nothing was changed") {
			msg += "; nothing was changed"
		}
	}
	fmt.Fprintln(os.Stderr, label+Wrap(msg, len(label), StderrWidth()))
	p.hint(e.TextHint())
	fmt.Fprintf(os.Stderr, "(code %s, exit %d)\n", e.Code, e.Exit)
}

func (p *Printer) hint(h string) {
	if h != "" {
		fmt.Fprintf(os.Stderr, "Hint: %s\n", Marks(Wrap(noteText(p.Redact(h)), len("Hint: "), StderrWidth())))
	}
}

func (p *Printer) writeJSON(v any) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if p.Pretty {
		enc.SetIndent("", "  ")
	}
	_ = enc.Encode(v)
	_, _ = p.W.Write(p.redactJSON(b.Bytes()))
}

func renderText(data any) string {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprint(data)
	}
	var v any
	_ = json.Unmarshal(b, &v)
	var sb strings.Builder
	writeAny(&sb, v, "")
	return sb.String()
}

func writeAny(sb *strings.Builder, v any, indent string) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if t[k] == nil {
				continue
			}
			switch c := t[k].(type) {
			case map[string]any, []any:
				fmt.Fprintf(sb, "%s%s:\n", indent, k)
				writeAny(sb, c, indent+"  ")
			default:
				fmt.Fprintf(sb, "%s%s: %v\n", indent, k, c)
			}
		}
	case []any:
		for _, item := range t {
			switch item.(type) {
			case map[string]any:
				var sub strings.Builder
				writeAny(&sub, item, indent+"  ")
				sb.WriteString(strings.Replace(sub.String(), indent+"  ", indent+"- ", 1))
			default:
				fmt.Fprintf(sb, "%s- %v\n", indent, item)
			}
		}
	default:
		fmt.Fprintf(sb, "%s%v\n", indent, t)
	}
}
