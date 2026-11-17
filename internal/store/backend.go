package store

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"

	"github.com/sgnd/kongkow/internal/sealdb"
	"github.com/sgnd/kongkow/internal/sealrow"
)

const StoreFile = "store.db"
const storeLock = ".store.lock"

var KeySource func(meta map[string]string, create bool) (master []byte, newMeta map[string]string, err error)

var ErrNoStore = errors.New("store: no store yet")

type openStore struct {
	db     *sealdb.DB
	fi     os.FileInfo
	master []byte
}

var (
	dbMu   sync.Mutex
	dbOpen = map[string]openStore{}
)

func DB(create bool) (*sealdb.DB, error) {
	d, err := cachedDB(create)
	if err != nil {
		return nil, err
	}
	if !errors.Is(d.View(context.Background(), func(*sealdb.Tx) error { return nil }), sealdb.ErrKeyChanged) {
		return d, nil
	}
	dbMu.Lock()
	if o, ok := dbOpen[Home()]; ok && o.db == d {
		d.Close()
		delete(dbOpen, Home())
	}
	dbMu.Unlock()
	return cachedDB(create)
}

func cachedDB(create bool) (*sealdb.DB, error) {
	dbMu.Lock()
	defer dbMu.Unlock()
	home := Home()
	if o, ok := dbOpen[home]; ok {
		if fi, err := os.Stat(DBPath()); err == nil && os.SameFile(fi, o.fi) {
			return o.db, nil
		}
		o.db.Close()
		delete(dbOpen, home)
	}
	if KeySource == nil {
		return nil, errors.New("store: no key source (the vault package is not linked)")
	}
	ctx := context.Background()
	path := DBPath()
	meta, err := sealdb.Peek(ctx, path)
	if errors.Is(err, os.ErrNotExist) {
		if err := missingStore(); err != nil {
			return nil, err
		}
		if !create {
			return nil, ErrNoStore
		}
		return createDB(ctx, home, path)
	}
	if err != nil {
		return nil, err
	}
	return openDB(ctx, home, path, meta)
}

func openDB(ctx context.Context, home, path string, meta map[string]string) (*sealdb.DB, error) {
	master, _, err := KeySource(meta, false)
	if err != nil {
		return nil, err
	}
	k, err := sealrow.NewKey(master)
	if err != nil {
		return nil, err
	}
	d, err := sealdb.Open(ctx, path, k)
	if err != nil {
		return nil, err
	}
	return remember(home, path, d, master)
}

func remember(home, path string, d *sealdb.DB, master []byte) (*sealdb.DB, error) {
	fi, err := os.Stat(path)
	if err != nil {
		d.Close()
		return nil, err
	}
	dbOpen[home] = openStore{d, fi, master}
	return d, nil
}

func createDB(ctx context.Context, home, path string) (*sealdb.DB, error) {
	if err := ensureHome(); err != nil {
		return nil, err
	}
	unlock, err := Lock(storeLock)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if meta, err := sealdb.Peek(ctx, path); err == nil {
		return openDB(ctx, home, path, meta)
	}
	master, meta, err := KeySource(nil, true)
	if err != nil {
		return nil, err
	}
	k, err := sealrow.NewKey(master)
	if err != nil {
		return nil, err
	}
	tmp := path + ".new"
	_ = os.Remove(tmp)
	d, err := sealdb.Open(ctx, tmp, k)
	if err == nil {
		err = d.Update(ctx, func(tx *sealdb.Tx) error {
			for key, v := range meta {
				if err := tx.SetMeta(key, v); err != nil {
					return err
				}
			}
			return nil
		})
		if cerr := d.Close(); err == nil {
			err = cerr
		}
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		os.Remove(tmp)
		return nil, err
	}
	d, err = sealdb.Open(ctx, path, k)
	if err != nil {
		return nil, err
	}
	return remember(home, path, d, master)
}

func Close() {
	dbMu.Lock()
	defer dbMu.Unlock()
	for h, o := range dbOpen {
		o.db.Close()
		delete(dbOpen, h)
	}
}

func MasterKey() ([]byte, error) {
	if _, err := DB(false); err != nil {
		return nil, err
	}
	dbMu.Lock()
	defer dbMu.Unlock()
	o, ok := dbOpen[Home()]
	if !ok {
		return nil, ErrNoStore
	}
	return bytes.Clone(o.master), nil
}

func CheckWritable() error {
	if _, err := DB(true); err != nil {
		return mapStoreErr(err)
	}
	f, err := os.OpenFile(DBPath(), os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	return f.Close()
}
