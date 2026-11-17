package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/sealdb"
	"github.com/sgnd/kongkow/internal/sealrow"
)

const kindSettings = "settings"

func mapStoreErr(err error) error {
	switch {
	case errors.Is(err, sealdb.ErrNewerFormat), errors.Is(err, sealrow.ErrNewerFormat):
		return out.Errf(out.ExitError, "STORE_NEWER", "a newer kongkow made or changed this store; update this one",
			"%s was made by a newer kongkow", Path(StoreFile))
	case errors.Is(err, sealdb.ErrKeyChanged):
		return out.Errf(out.ExitError, "STORE_KEY_CHANGED", "run it again: another kongkow changed the store's key while this one ran",
			"%s: %v", Path(StoreFile), err)
	case errors.Is(err, sealdb.ErrTampered), errors.Is(err, sealdb.ErrWrongKey), errors.Is(err, sealrow.ErrOpen):
		return out.Errf(out.ExitError, "STATE_CORRUPT",
			"the store does not open with this key or was changed outside kongkow; restore the kongkow folder from a backup",
			"%s: %v", Path(StoreFile), err)
	}
	return err
}

func loadSealed(tx *sealdb.Tx) (*State, error) {
	s := &State{Settings: DefaultSettings()}
	s.init()
	put := func(kind string, r sealdb.Record) error {
		var name string
		var rec *Rec
		switch kind {
		case KindPost:
			v := &Post{}
			if err := json.Unmarshal(r.Data, v); err != nil {
				return err
			}
			if s.Posts[v.Name] != nil {
				return dupName(kind, v.Name)
			}
			s.Posts[v.Name], name, rec = v, v.Name, &v.Rec
		case KindDest:
			v := &Dest{}
			if err := json.Unmarshal(r.Data, v); err != nil {
				return err
			}
			if s.Dests[v.Name] != nil {
				return dupName(kind, v.Name)
			}
			s.Dests[v.Name], name, rec = v, v.Name, &v.Rec
		case KindCollection:
			v := &Collection{}
			if err := json.Unmarshal(r.Data, v); err != nil {
				return err
			}
			if s.Collections[v.Name] != nil {
				return dupName(kind, v.Name)
			}
			s.Collections[v.Name], name, rec = v, v.Name, &v.Rec
		case KindBookmark:
			v := &Bookmark{}
			if err := json.Unmarshal(r.Data, v); err != nil {
				return err
			}
			if s.Bookmarks[v.Name] != nil {
				return dupName(kind, v.Name)
			}
			s.Bookmarks[v.Name], name, rec = v, v.Name, &v.Rec
		case KindJob:
			v := &Job{}
			if err := json.Unmarshal(r.Data, v); err != nil {
				return err
			}
			if v.ID == "" {
				return fmt.Errorf("a job record without an id: %w", sealdb.ErrTampered)
			}
			if s.Jobs[v.ID] != nil {
				return dupName(kind, v.ID)
			}
			s.Jobs[v.ID], name, rec = v, v.ID, &v.Rec
		}
		if name == "" {
			return fmt.Errorf("a %s record without a name: %w", kind, sealdb.ErrTampered)
		}
		*rec = Rec{ID: r.ID, Version: r.Version}
		return nil
	}
	for _, kind := range []string{KindPost, KindDest, KindCollection, KindBookmark, KindJob} {
		rows, err := tx.List(kind)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			if err := put(kind, r); err != nil {
				return nil, err
			}
		}
	}
	rows, err := tx.List(kindSettings)
	if err != nil {
		return nil, err
	}
	switch len(rows) {
	case 0:
	case 1:
		if err := json.Unmarshal(rows[0].Data, &s.Settings); err != nil {
			return nil, err
		}
		s.settingsRec = Rec{ID: rows[0].ID, Version: rows[0].Version}
	default:
		return nil, fmt.Errorf("%d settings records: %w", len(rows), sealdb.ErrTampered)
	}
	s.init()
	return s, nil
}

func dupName(kind, name string) error {
	return fmt.Errorf("two %s records named %q: %w", kind, name, sealdb.ErrTampered)
}

func Load() (*State, error) {
	if err := ensureHome(); err != nil {
		return nil, err
	}
	d, err := DB(false)
	if errors.Is(err, ErrNoStore) {
		s := &State{Settings: DefaultSettings()}
		s.init()
		return s, nil
	}
	if err != nil {
		return nil, mapStoreErr(err)
	}
	var s *State
	err = d.View(context.Background(), func(tx *sealdb.Tx) error {
		var err error
		s, err = loadSealed(tx)
		return err
	})
	if err != nil {
		return nil, mapStoreErr(err)
	}
	return s, nil
}

func updateStore(fn func(*State) error) error {
	if err := ensureHome(); err != nil {
		return err
	}
	d, err := DB(true)
	if err != nil {
		return mapStoreErr(err)
	}
	var s *State
	err = d.Update(context.Background(), func(tx *sealdb.Tx) error {
		var err error
		if s, err = loadSealed(tx); err != nil {
			return err
		}
		before := s.recSnapshot()
		setBefore, err := json.Marshal(&s.Settings)
		if err != nil {
			return err
		}
		if err := fn(s); err != nil {
			return err
		}
		return s.writeSealed(tx, before, setBefore)
	})
	if err != nil {
		var oe *out.Error
		if errors.As(err, &oe) {
			return err
		}
		return mapStoreErr(err)
	}
	return nil
}

func (s *State) writeSealed(tx *sealdb.Tx, before map[string]recSnap, setBefore []byte) error {
	if err := s.bumpRecs(before); err != nil {
		return err
	}
	kept := map[string]bool{}
	for _, r := range s.recs() {
		kept[r.rec.ID] = true
		old, had := before[r.rec.ID]
		b, err := json.Marshal(r.v)
		if err != nil {
			return err
		}
		if had && old.kind == r.kind && old.name == r.name && bytes.Equal(old.body, b) {
			r.rec.Version = old.rec.Version
			continue
		}
		rec, err := tx.Put(r.kind, r.rec.ID, b)
		if err != nil {
			return err
		}
		r.rec.Version = rec.Version
	}
	for id := range before {
		if !kept[id] {
			if err := tx.Delete(id); err != nil && !errors.Is(err, sealdb.ErrNotFound) {
				return err
			}
		}
	}
	b, err := json.Marshal(&s.Settings)
	if err != nil {
		return err
	}
	if s.settingsRec.ID == "" || !bytes.Equal(b, setBefore) {
		rec, err := tx.Put(kindSettings, s.settingsRec.ID, b)
		if err != nil {
			return err
		}
		s.settingsRec = Rec{ID: rec.ID, Version: rec.Version}
	}
	return nil
}
