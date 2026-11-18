package vault

import (
	"errors"
	"testing"

	"github.com/sgnd/kongkow/internal/store"
)

func freshHome(t *testing.T) {
	t.Helper()
	t.Setenv("KONGKOW_HOME", t.TempDir())
	store.Close()
	t.Cleanup(store.Close)
}

func TestSetGetDelete(t *testing.T) {
	freshHome(t)
	if err := Set("tg", "s3cret"); err != nil {
		t.Fatal(err)
	}
	v, err := Get("tg")
	if err != nil || v != "s3cret" {
		t.Fatalf("get %q %v", v, err)
	}
	if v, err := Resolve("secret:tg"); err != nil || v != "s3cret" {
		t.Fatalf("resolve %q %v", v, err)
	}
	infos, err := List()
	if err != nil || len(infos) != 1 {
		t.Fatalf("list %v %v", infos, err)
	}
	if err := Delete("tg"); err != nil {
		t.Fatal(err)
	}
	if _, err := Get("tg"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	freshHome(t)
	store.DryRun = true
	t.Cleanup(func() { store.DryRun = false })
	if err := Set("tg", "x"); !errors.Is(err, store.ErrDryRun) {
		t.Fatalf("want ErrDryRun, got %v", err)
	}
}

func TestValidRef(t *testing.T) {
	if !ValidRef("secret:tg") {
		t.Fatal("secret:tg should be valid")
	}
	if ValidRef("tg") || ValidRef("") {
		t.Fatal("bare names must be refused")
	}
}
