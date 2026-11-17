package vault

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/crypto/scrypt"

	"github.com/sgnd/kongkow/internal/store"
	"github.com/sgnd/kongkow/internal/tmpclean"
)

func init() { store.KeySource = storeMaster }

func storeKeyFile() string { return store.Path("store.key") }

func storeMaster(meta map[string]string, create bool) ([]byte, map[string]string, error) {
	if create {
		return newStoreMaster()
	}
	switch meta["key_mode"] {
	case "passphrase":
		p := Passphrase()
		if p == "" {
			return nil, nil, ErrLocked
		}
		salt, err := hex.DecodeString(meta["salt"])
		n, nerr := strconv.Atoi(meta["scrypt_n"])
		if err != nil || len(salt) < 16 || nerr != nil || n < 2 {
			return nil, nil, fmt.Errorf("%w: the store's passphrase settings are damaged", ErrKey)
		}
		k, err := scrypt.Key([]byte(p), salt, n, 8, 1, keyLen)
		return k, nil, err
	case "keyfile":
		kp := storeKeyFile()
		b, err := os.ReadFile(kp)
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("%w: %s is missing, and the store cannot be opened without it; restore it from a backup", ErrKey, kp)
		}
		if err != nil {
			return nil, nil, err
		}
		if len(b) < keyLen {
			return nil, nil, fmt.Errorf("%w: %s holds %d bytes, not %d", ErrKey, kp, len(b), keyLen)
		}
		return b[:keyLen], nil, nil
	case "keychain":
		// Scaffold: OS keychain support is not wired yet; use a key file or passphrase.
		return nil, nil, fmt.Errorf("%w: this store uses the OS keychain; set KONGKOW_VAULT_PASS after migrating, or restore a keyfile store", ErrKey)
	}
	return nil, nil, fmt.Errorf("%w: the store names a key mode this kongkow does not know (%q); update kongkow", ErrKey, meta["key_mode"])
}

func newStoreMaster() ([]byte, map[string]string, error) {
	if p := Passphrase(); p != "" {
		salt := make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return nil, nil, err
		}
		k, err := scrypt.Key([]byte(p), salt, PassphraseN, 8, 1, keyLen)
		if err != nil {
			return nil, nil, err
		}
		return k, map[string]string{"key_mode": "passphrase", "salt": hex.EncodeToString(salt), "scrypt_n": strconv.Itoa(PassphraseN)}, nil
	}
	if err := store.EnsureHome(); err != nil {
		return nil, nil, err
	}
	k, err := newKeyFile(storeKeyFile())
	if err != nil {
		return nil, nil, err
	}
	if len(k) < keyLen {
		return nil, nil, fmt.Errorf("%w: %s holds %d bytes, not %d", ErrKey, storeKeyFile(), len(k), keyLen)
	}
	return k[:keyLen], map[string]string{"key_mode": "keyfile"}, nil
}

func newKeyFile(kp string) ([]byte, error) {
	key := make([]byte, keyLen)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(kp), ".tmp-key-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	defer tmpclean.Add(tmp.Name())()
	_, err = tmp.Write(key)
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}
	switch err = os.Link(tmp.Name(), kp); {
	case err == nil:
		return key, nil
	case errors.Is(err, os.ErrExist):
		return os.ReadFile(kp)
	}
	f, err := os.OpenFile(kp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return os.ReadFile(kp)
	}
	if err != nil {
		return nil, err
	}
	_, err = f.Write(key)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(kp)
		return nil, err
	}
	return key, nil
}
