package cli

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sgnd/kongkow/internal/dest"
	"github.com/sgnd/kongkow/internal/store"
)

func TestApplyResultKeepsTheLiveURL(t *testing.T) {
	j := &store.Job{ID: "j1", Status: store.JobRunning}
	applyResult(j, &dest.Result{Detail: "posted https://m.example/@me/1", URL: "https://m.example/@me/1"}, nil, time.Unix(100, 0))
	if j.Status != store.JobDone || j.URL != "https://m.example/@me/1" || j.Result != "posted https://m.example/@me/1" {
		t.Errorf("done job: %+v", j)
	}

	f := &store.Job{ID: "j2", Status: store.JobRunning, URL: "stale"}
	applyResult(f, nil, errors.New("mastodon: 401"), time.Unix(100, 0))
	if f.Status != store.JobFailed || f.URL != "" || f.Result != "mastodon: 401" {
		t.Errorf("failed job: %+v", f)
	}

	p := &store.Job{ID: "j3"}
	applyResult(p, &dest.Result{Path: "/out/a.md", Detail: "wrote file"}, nil, time.Unix(100, 0))
	if p.Result != "/out/a.md" || p.URL != "" {
		t.Errorf("file job: %+v", p)
	}
}

func TestStatusTextShowsURLAndFailure(t *testing.T) {
	r := statusResult{Jobs: []jobRow{
		{ID: "a", Post: "p", Dest: "m", At: time.Now(), Status: store.JobDone, URL: "https://m.example/@me/1"},
		{ID: "b", Post: "p", Dest: "m", At: time.Now(), Status: store.JobFailed, Result: "mastodon: 401"},
		{ID: "c", Post: "p", Dest: "f", At: time.Now(), Status: store.JobDone, Result: "/out/a.md"},
	}}
	got := r.Text()
	for _, want := range []string{"https://m.example/@me/1", "mastodon: 401", "/out/a.md"} {
		if !strings.Contains(got, want) {
			t.Errorf("status text lacks %q:\n%s", want, got)
		}
	}
}

func TestStatusTextDropsControlCharactersFromRemoteText(t *testing.T) {
	r := statusResult{Jobs: []jobRow{{ID: "a", Post: "p", Dest: "m", At: time.Now(), Status: store.JobDone, URL: "https://x.example/\x1b[2Jhi"}}}
	if strings.Contains(r.Text(), "\x1b") {
		t.Error("a control character from a server reached the terminal")
	}
}

func TestPublishedRowHasURL(t *testing.T) {
	row := publishedRow("p", &store.Dest{Name: "m", Kind: store.DestMastodon}, &dest.Result{URL: "https://m.example/@me/1", Detail: "posted"})
	if row["url"] != "https://m.example/@me/1" {
		t.Errorf("row %v", row)
	}
	if _, ok := publishedRow("p", &store.Dest{Name: "f", Kind: store.DestFile}, &dest.Result{Path: "/x"})["url"]; ok {
		t.Error("a file destination has no url key")
	}
}
