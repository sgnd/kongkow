package store

import (
	"errors"
	"testing"
	"time"
)

func freshHome(t *testing.T) {
	t.Helper()
	old := KeySource
	KeySource = func(map[string]string, bool) ([]byte, map[string]string, error) {
		return make([]byte, 32), map[string]string{}, nil
	}
	t.Cleanup(func() { KeySource = old })
	t.Setenv("KONGKOW_HOME", t.TempDir())
	Close()
	t.Cleanup(Close)
}

func TestUpdateThenLoad(t *testing.T) {
	freshHome(t)
	now := time.Now().UTC()
	if err := Update(func(s *State) error {
		s.Posts["b"] = &Post{Name: "b", Body: "two", CreatedAt: now, UpdatedAt: now}
		s.Posts["a"] = &Post{Name: "a", Body: "one", CreatedAt: now, UpdatedAt: now}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	Close() // read it back from disk, not from memory
	st, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	ps := st.SortedPosts()
	if len(ps) != 2 || ps[0].Name != "a" || ps[1].Body != "two" {
		t.Fatalf("posts %+v", ps)
	}
}

func TestUpdateErrorKeepsState(t *testing.T) {
	freshHome(t)
	boom := errors.New("boom")
	err := Update(func(s *State) error {
		s.Posts["x"] = &Post{Name: "x"}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err %v", err)
	}
	st, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if st.Posts["x"] != nil {
		t.Fatal("failed update was saved")
	}
}
