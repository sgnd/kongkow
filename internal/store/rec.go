package store

// Rec is the sealed-store identity of an inventory item.
type Rec struct {
	ID      string
	Version uint64
}

type recSnap struct {
	kind string
	name string
	rec  Rec
	body []byte
}

type recRef struct {
	kind string
	name string
	rec  *Rec
	v    any
}

func (s *State) recs() []recRef {
	out := make([]recRef, 0, len(s.Posts)+len(s.Dests)+len(s.Collections)+len(s.Bookmarks)+len(s.Jobs))
	for _, p := range s.Posts {
		out = append(out, recRef{KindPost, p.Name, &p.Rec, p})
	}
	for _, d := range s.Dests {
		out = append(out, recRef{KindDest, d.Name, &d.Rec, d})
	}
	for _, c := range s.Collections {
		out = append(out, recRef{KindCollection, c.Name, &c.Rec, c})
	}
	for _, b := range s.Bookmarks {
		out = append(out, recRef{KindBookmark, b.Name, &b.Rec, b})
	}
	for _, j := range s.Jobs {
		out = append(out, recRef{KindJob, j.ID, &j.Rec, j})
	}
	return out
}

func (s *State) recSnapshot() map[string]recSnap {
	m := map[string]recSnap{}
	for _, r := range s.recs() {
		if r.rec.ID == "" {
			continue
		}
		b, _ := jsonMarshal(r.v)
		m[r.rec.ID] = recSnap{kind: r.kind, name: r.name, rec: *r.rec, body: b}
	}
	return m
}

func (s *State) bumpRecs(before map[string]recSnap) error {
	for _, r := range s.recs() {
		if r.rec.ID == "" {
			id, err := newID()
			if err != nil {
				return err
			}
			r.rec.ID = id
		}
	}
	return nil
}
