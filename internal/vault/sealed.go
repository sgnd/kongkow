package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/sgnd/kongkow/internal/sealdb"
	"github.com/sgnd/kongkow/internal/store"
)

const (
	secretType     = "secret"
	vaultHKDFLabel = "kongkow vault entries v1"
)

type sealedEntry struct {
	Name      string    `json:"name"`
	Nonce     []byte    `json:"nonce"`
	Data      []byte    `json:"data"`
	UpdatedAt time.Time `json:"updated_at"`
}

func loadSealedTx(tx *sealdb.Tx) (*file, error) {
	f := &file{Entries: map[string]*entry{}}
	rows, err := tx.List(secretType)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		var e sealedEntry
		if err := json.Unmarshal(r.Data, &e); err != nil {
			return nil, err
		}
		if e.Name == "" || f.Entries[e.Name] != nil {
			return nil, sealdb.ErrTampered
		}
		f.Entries[e.Name] = &entry{Nonce: e.Nonce, Data: e.Data, UpdatedAt: e.UpdatedAt, id: r.ID}
	}
	return f, nil
}

func loadSealed() (*file, error) {
	d, err := store.DB(false)
	if errors.Is(err, store.ErrNoStore) {
		return &file{Entries: map[string]*entry{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var f *file
	err = d.View(context.Background(), func(tx *sealdb.Tx) error {
		var err error
		f, err = loadSealedTx(tx)
		return err
	})
	return f, err
}

func changeSealed(fn func(f *file) error) error {
	d, err := store.DB(true)
	if err != nil {
		return err
	}
	return d.Update(context.Background(), func(tx *sealdb.Tx) error {
		f, err := loadSealedTx(tx)
		if err != nil {
			return err
		}
		before := map[string]entry{}
		for _, e := range f.Entries {
			before[e.id] = *e
		}
		if err := fn(f); err != nil {
			return err
		}
		kept := map[string]bool{}
		for name, e := range f.Entries {
			old, had := before[e.id]
			if had && e.id != "" && !kept[e.id] && string(old.Nonce) == string(e.Nonce) && string(old.Data) == string(e.Data) && old.UpdatedAt.Equal(e.UpdatedAt) {
				kept[e.id] = true
				continue
			}
			id := e.id
			if kept[id] {
				id = ""
			}
			b, err := json.Marshal(sealedEntry{Name: name, Nonce: e.Nonce, Data: e.Data, UpdatedAt: e.UpdatedAt})
			if err != nil {
				return err
			}
			r, err := tx.Put(secretType, id, b)
			if err != nil {
				return err
			}
			e.id = r.ID
			kept[r.ID] = true
		}
		for id := range before {
			if !kept[id] {
				if err := tx.Delete(id); err != nil && !errors.Is(err, sealdb.ErrNotFound) {
					return err
				}
			}
		}
		return nil
	})
}

func sealedAEAD() (cipher.AEAD, error) {
	m, err := store.MasterKey()
	if err != nil {
		return nil, err
	}
	defer clear(m)
	k, err := hkdf.Key(sha256.New, m, nil, vaultHKDFLabel, 32)
	if err != nil {
		return nil, err
	}
	defer clear(k)
	blk, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(blk)
}

func sealedStrength() (Strength, error) {
	meta, err := sealdb.Peek(context.Background(), store.DBPath())
	if errors.Is(err, os.ErrNotExist) {
		return Strength{NoVaultYet: true}, nil
	}
	if err != nil {
		return Strength{}, err
	}
	s := Strength{Mode: meta["key_mode"]}
	if n, err := strconv.Atoi(meta["scrypt_n"]); err == nil {
		s.ScryptN = n
	}
	if s.Mode == "passphrase" {
		s.OldKDF = s.ScryptN < PassphraseN
		s.ShortPass = len([]rune(Passphrase())) < MinPassphrase
	}
	return s, nil
}

func clear(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
