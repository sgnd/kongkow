package sealdb

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

func scrub(b []byte) []byte { return bytes.ReplaceAll(b, []byte("SECRET"), []byte("<redacted>")) }

func TestRewriteAuditKeepsTheChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.db")
	d := openTest(t, path)
	appendN(t, d, "one", "two SECRET here", "three", "four SECRET")

	var n int
	if err := d.Update(context.Background(), func(tx *Tx) error {
		var err error
		n, err = tx.RewriteAudit(scrub)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("rewrote %d entries, want 2", n)
	}
	appendN(t, d, "five") // the head must still fit
	d.Close()

	d = openTest(t, path) // a fresh open checks the chain and the head again
	defer d.Close()
	rows, err := readAudit(d)
	if err != nil {
		t.Fatalf("chain broken after the rewrite: %v", err)
	}
	want := []string{"one", "two <redacted> here", "three", "four <redacted>", "five"}
	if len(rows) != len(want) {
		t.Fatalf("rows %d, want %d", len(rows), len(want))
	}
	for i, r := range rows {
		if string(r.Data) != want[i] || r.Seq != int64(i+1) {
			t.Errorf("row %d: seq %d data %q, want %q", i, r.Seq, r.Data, want[i])
		}
	}
}

func TestRewriteAuditWithNothingToChangeWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.db")
	d := openTest(t, path)
	defer d.Close()
	appendN(t, d, "one", "two")
	before, err := d.Generation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err := d.Update(context.Background(), func(tx *Tx) error {
		var err error
		n, err = tx.RewriteAudit(scrub)
		return err
	}); err != nil || n != 0 {
		t.Fatalf("n %d err %v", n, err)
	}
	after, err := d.Generation(context.Background())
	if err != nil || before != after {
		t.Errorf("generation moved from %d to %d with nothing rewritten", before, after)
	}
}

func TestRewriteAuditNeedsAWriteStep(t *testing.T) {
	d := openTest(t, filepath.Join(t.TempDir(), "store.db"))
	defer d.Close()
	appendN(t, d, "one")
	err := d.View(context.Background(), func(tx *Tx) error { _, err := tx.RewriteAudit(scrub); return err })
	if err != ErrReadOnly {
		t.Errorf("err %v, want ErrReadOnly", err)
	}
}
