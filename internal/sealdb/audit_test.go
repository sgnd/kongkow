package sealdb

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/sgnd/kongkow/internal/sealrow"
)

func openTest(t *testing.T, path string) *DB {
	t.Helper()
	k, err := sealrow.NewKey(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	d, err := Open(context.Background(), path, k)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func appendN(t *testing.T, d *DB, msgs ...string) {
	t.Helper()
	for _, m := range msgs {
		if err := d.Update(context.Background(), func(tx *Tx) error { return tx.AppendAudit([]byte(m)) }); err != nil {
			t.Fatal(err)
		}
	}
}

func readAudit(d *DB) ([]AuditRow, error) {
	var rows []AuditRow
	err := d.View(context.Background(), func(tx *Tx) error {
		var err error
		rows, err = tx.Audit(0, 100)
		return err
	})
	return rows, err
}

func TestAuditRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.db")
	d := openTest(t, path)
	appendN(t, d, "one", "two", "three")
	d.Close()

	d = openTest(t, path)
	defer d.Close()
	rows, err := readAudit(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || string(rows[0].Data) != "one" || string(rows[2].Data) != "three" {
		t.Fatalf("rows %+v", rows)
	}
	for i, r := range rows {
		if r.Seq != int64(i+1) {
			t.Fatalf("row %d seq %d", i, r.Seq)
		}
	}
}

func TestAuditDeletedRowIsTampered(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.db")
	d := openTest(t, path)
	appendN(t, d, "one", "two", "three")
	d.Close()

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`DELETE FROM audit WHERE seq = 2`); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	d = openTest(t, path)
	defer d.Close()
	if _, err := readAudit(d); !errors.Is(err, ErrTampered) {
		t.Fatalf("want ErrTampered, got %v", err)
	}
}
