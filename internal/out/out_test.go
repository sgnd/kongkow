package out

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: "json", W: &buf, Version: "test"}
	p.Print("version", map[string]any{"version": "test"}, nil)
	var env Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if !env.OK || env.Command != "version" || env.Meta.Contract != 1 {
		t.Fatalf("%+v", env)
	}
}

func TestFailUsage(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: "json", W: &buf, Version: "test"}
	code := p.Fail("list", Usage("need a name"))
	if code != ExitUsage {
		t.Fatalf("exit %d", code)
	}
}
