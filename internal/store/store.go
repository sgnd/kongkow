// Package store keeps kongkow state (posts, destinations, collections,
// bookmarks, jobs, settings) in the sealed store under $KONGKOW_HOME
// (default ~/.kongkow).
package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/sgnd/kongkow/internal/flock"
	"github.com/sgnd/kongkow/internal/tmpclean"
)

const (
	KindPost       = "post"
	KindDraft      = KindPost // alias kept for older comments
	KindDest       = "dest"
	KindCollection = "collection"
	KindBookmark   = "bookmark"
	KindJob        = "job"
)

// Post is local content waiting to be published.
type Post struct {
	Name       string            `json:"name"`
	Title      string            `json:"title,omitempty"`
	Body       string            `json:"body,omitempty"`
	Media      []string          `json:"media,omitempty"` // local file paths
	Alt        []string          `json:"alt,omitempty"`   // alt text per media file, same order
	Collection string            `json:"collection,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Notes      string            `json:"notes,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Rec        Rec               `json:"-"`
}

// Draft is an alias name used in early scaffold docs; prefer Post.
type Draft = Post

// Dest is a publish destination (Telegram, file sink, …).
type Dest struct {
	Name      string            `json:"name"`
	Kind      string            `json:"kind"` // telegram | mastodon | bluesky | file
	Target    string            `json:"target,omitempty"`
	SecretRef string            `json:"secret_ref,omitempty"` // secret:NAME
	Options   map[string]string `json:"options,omitempty"`
	Protected bool              `json:"protected,omitempty"`
	Slots     []string          `json:"slots,omitempty"` // daily post times, HH:MM local, sorted
	Tags      []string          `json:"tags,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Notes     string            `json:"notes,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Rec       Rec               `json:"-"`
}

const (
	DestTelegram = "telegram"
	DestFile     = "file"
	DestMastodon = "mastodon"
	DestBluesky  = "bluesky"
)

var DestKinds = []string{DestTelegram, DestMastodon, DestBluesky, DestFile}

// Collection groups posts.
type Collection struct {
	Name      string            `json:"name"`
	Title     string            `json:"title,omitempty"`
	Notes     string            `json:"notes,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Rec       Rec               `json:"-"`
}

// Bookmark is a saved URL or reference.
type Bookmark struct {
	Name      string            `json:"name"`
	URL       string            `json:"url"`
	Title     string            `json:"title,omitempty"`
	Notes     string            `json:"notes,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Rec       Rec               `json:"-"`
}

// Job is a local scheduled publish.
type Job struct {
	ID        string    `json:"id"`
	Post      string    `json:"post"`
	Dest      string    `json:"dest"`
	At        time.Time `json:"at"`
	Status    string    `json:"status"` // pending | done | failed | cancelled
	Result    string    `json:"result,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Rec       Rec       `json:"-"`
}

const (
	JobPending   = "pending"
	JobRunning   = "running"
	JobDone      = "done"
	JobFailed    = "failed"
	JobCancelled = "cancelled"
)

type Settings struct {
	Audit bool `json:"audit"`
}

func DefaultSettings() Settings {
	return Settings{Audit: true}
}

type State struct {
	Posts       map[string]*Post       `json:"posts"`
	Drafts      map[string]*Post       `json:"-"` // mirrors Posts for list helpers
	Dests       map[string]*Dest       `json:"dests"`
	Collections map[string]*Collection `json:"collections"`
	Bookmarks   map[string]*Bookmark   `json:"bookmarks"`
	Jobs        map[string]*Job        `json:"jobs"`
	Settings    Settings               `json:"settings"`

	settingsRec Rec
}

func Home() string {
	if h := os.Getenv("KONGKOW_HOME"); h != "" {
		return h
	}
	d, err := os.UserHomeDir()
	if err != nil {
		return ".kongkow"
	}
	return filepath.Join(d, ".kongkow")
}

func Path(parts ...string) string { return filepath.Join(append([]string{Home()}, parts...)...) }

func EnsureHome() error { return ensureHome() }

func ensureHome() error {
	h := Home()
	made := missingFolders(h)
	if err := os.MkdirAll(h, 0o700); err != nil {
		return err
	}
	for _, d := range made {
		if err := makePrivate(d); err != nil {
			for i := len(made) - 1; i >= 0; i-- {
				_ = os.Remove(made[i])
			}
			return fmt.Errorf("could not make %s private to you (%w); set KONGKOW_HOME to a folder on an NTFS drive", d, err)
		}
	}
	if len(made) == 0 {
		tightenHome(h)
	}
	return nil
}

var makePrivate = privateHome

func missingFolders(h string) []string {
	var made []string
	for d := h; ; {
		if _, err := os.Stat(d); !errors.Is(err, fs.ErrNotExist) {
			break
		}
		made = append([]string{d}, made...)
		up := filepath.Dir(d)
		if up == d {
			break
		}
		d = up
	}
	return made
}

func (s *State) init() {
	if s.Posts == nil {
		s.Posts = map[string]*Post{}
	}
	s.Drafts = s.Posts
	if s.Dests == nil {
		s.Dests = map[string]*Dest{}
	}
	if s.Collections == nil {
		s.Collections = map[string]*Collection{}
	}
	if s.Bookmarks == nil {
		s.Bookmarks = map[string]*Bookmark{}
	}
	if s.Jobs == nil {
		s.Jobs = map[string]*Job{}
	}
}

var DryRun bool
var ErrDryRun = errors.New("this command cannot preview its change; nothing was changed")

func Update(fn func(s *State) error) error {
	if DryRun {
		s, err := Load()
		if err != nil {
			return err
		}
		return fn(s)
	}
	return updateStore(fn)
}

func Lock(name string) (unlock func(), err error) {
	if err := ensureHome(); err != nil {
		return nil, err
	}
	lf, err := os.OpenFile(Path(name), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := flock.Lock(lf); err != nil {
		lf.Close()
		return nil, err
	}
	return func() { _ = flock.Unlock(lf); lf.Close() }, nil
}

func WriteFileAtomic(path string, b []byte, mode os.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer tmpclean.Add(tmp.Name())()
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return nil
	}
	defer d.Close()
	_ = d.Sync()
	return nil
}

func (s *State) SortedPosts() []*Post {
	ps := make([]*Post, 0, len(s.Posts))
	for _, p := range s.Posts {
		ps = append(ps, p)
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].Name < ps[j].Name })
	return ps
}

func (s *State) SortedDrafts() []*Post { return s.SortedPosts() }

func (s *State) SortedDests() []*Dest {
	ds := make([]*Dest, 0, len(s.Dests))
	for _, d := range s.Dests {
		ds = append(ds, d)
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i].Name < ds[j].Name })
	return ds
}

func (s *State) SortedCollections() []*Collection {
	cs := make([]*Collection, 0, len(s.Collections))
	for _, c := range s.Collections {
		cs = append(cs, c)
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
	return cs
}

func (s *State) SortedBookmarks() []*Bookmark {
	bs := make([]*Bookmark, 0, len(s.Bookmarks))
	for _, b := range s.Bookmarks {
		bs = append(bs, b)
	}
	sort.Slice(bs, func(i, j int) bool { return bs[i].Name < bs[j].Name })
	return bs
}

func (s *State) SortedJobs() []*Job {
	js := make([]*Job, 0, len(s.Jobs))
	for _, j := range s.Jobs {
		js = append(js, j)
	}
	sort.Slice(js, func(i, j int) bool {
		if js[i].At.Equal(js[j].At) {
			return js[i].ID < js[j].ID
		}
		return js[i].At.Before(js[j].At)
	})
	return js
}
