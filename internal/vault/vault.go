// Package vault stores secrets encrypted with AES-256-GCM, each bound to its
// name. The key is derived with scrypt from $KONGKOW_VAULT_PASS when set,
// otherwise from the sealed store's key (key file beside the store).
package vault

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sgnd/kongkow/internal/store"
)

type file struct {
	Entries map[string]*entry
}

var PassphraseN = 1 << 17

type entry struct {
	Nonce     []byte    `json:"nonce"`
	Data      []byte    `json:"data"`
	UpdatedAt time.Time `json:"updated_at"`
	id        string
}

type Info struct {
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	passMu sync.Mutex
	pass   string
)

func Passphrase() string {
	passMu.Lock()
	defer passMu.Unlock()
	if p, ok := os.LookupEnv("KONGKOW_VAULT_PASS"); ok {
		pass = p
		_ = os.Unsetenv("KONGKOW_VAULT_PASS")
	}
	return pass
}

func TakeUnlockEnv() { Passphrase() }

func load() (*file, error) {
	if err := store.EnsureHome(); err != nil {
		return nil, err
	}
	return loadSealed()
}

func (f *file) aead() (cipher.AEAD, error) { return sealedAEAD() }

const keyLen = 32

func change(fn func(f *file) error) error {
	if store.DryRun {
		return store.ErrDryRun
	}
	return changeSealed(fn)
}

type Strength struct {
	Mode       string `json:"mode"`
	ScryptN    int    `json:"scrypt_n"`
	ShortPass  bool   `json:"short_passphrase,omitempty"`
	OldKDF     bool   `json:"old_kdf,omitempty"`
	NoVaultYet bool   `json:"no_vault_yet,omitempty"`
}

const MinPassphrase = 15

func Check() (Strength, error) { return sealedStrength() }

func Set(name, value string) error {
	return change(func(f *file) error {
		a, err := f.aead()
		if err != nil {
			return err
		}
		if err := f.checkKey(a); err != nil {
			return err
		}
		nonce := make([]byte, a.NonceSize())
		if _, err := rand.Read(nonce); err != nil {
			return err
		}
		f.Entries[name] = &entry{Nonce: nonce, Data: a.Seal(nil, nonce, []byte(value), []byte(name)), UpdatedAt: time.Now().UTC()}
		return nil
	})
}

func (f *file) checkKey(a cipher.AEAD) error {
	for name, e := range f.Entries {
		if _, err := a.Open(nil, e.Nonce, e.Data, []byte(name)); err != nil {
			return fmt.Errorf("%w %q (wrong KONGKOW_VAULT_PASS or store.key?): nothing was saved", ErrDecrypt, name)
		}
		return nil
	}
	return nil
}

var ErrNotFound = errors.New("secret not found")

var (
	ErrLocked  = errors.New("the vault is locked with a passphrase; set KONGKOW_VAULT_PASS")
	ErrKey     = errors.New("the vault's key cannot be used")
	ErrDecrypt = errors.New("cannot decrypt secret")
)

func Get(name string) (string, error) {
	f, err := load()
	if err != nil {
		return "", err
	}
	e, ok := f.Entries[name]
	if !ok {
		return "", ErrNotFound
	}
	a, err := f.aead()
	if err != nil {
		return "", err
	}
	pt, err := a.Open(nil, e.Nonce, e.Data, []byte(name))
	if err != nil {
		return "", fmt.Errorf("%w %q (wrong KONGKOW_VAULT_PASS or store.key?)", ErrDecrypt, name)
	}
	return string(pt), nil
}

func Delete(name string) error {
	return change(func(f *file) error {
		if _, ok := f.Entries[name]; !ok {
			return ErrNotFound
		}
		delete(f.Entries, name)
		return nil
	})
}

func List() ([]Info, error) {
	f, err := load()
	if err != nil {
		return nil, err
	}
	out := []Info{}
	for n, e := range f.Entries {
		out = append(out, Info{Name: n, UpdatedAt: e.UpdatedAt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

var Refs = []string{
	"secret:NAME (kongkow's vault)",
	"env:VAR",
	"file:PATH",
}

func ValidRef(ref string) bool {
	for _, p := range []string{"secret:", "env:", "file:"} {
		if strings.HasPrefix(ref, p) && len(ref) > len(p) {
			return true
		}
	}
	return false
}

func RefHelp() string { return "use one of: " + strings.Join(Refs, ", ") }

func Resolve(ref string) (string, error) {
	switch {
	case ref == "":
		return "", nil
	case strings.HasPrefix(ref, "secret:"):
		return Get(strings.TrimPrefix(ref, "secret:"))
	case strings.HasPrefix(ref, "env:"):
		name := strings.TrimPrefix(ref, "env:")
		v, ok := os.LookupEnv(name)
		if !ok {
			return "", fmt.Errorf("environment variable %s not set", name)
		}
		return v, nil
	case strings.HasPrefix(ref, "file:"):
		b, err := os.ReadFile(strings.TrimPrefix(ref, "file:"))
		return strings.TrimRight(string(b), "\r\n"), err
	}
	return "", fmt.Errorf("%q is not a secret reference (%s)", ref, RefHelp())
}
