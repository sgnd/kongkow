package store

import (
	"os"

	"github.com/sgnd/kongkow/internal/out"
)

func DBPath() string { return Path(StoreFile) }

func sealedSign() string {
	for _, f := range []string{"store.key"} {
		if _, err := os.Stat(Path(f)); err == nil {
			return f
		}
	}
	return ""
}

func missingStore() error {
	sign := sealedSign()
	if sign == "" {
		return nil
	}
	return out.Errf(out.ExitError, "STATE_CORRUPT",
		"put store.db back from a backup",
		"%s is missing, though this folder had a store (%s)", Path(StoreFile), sign)
}
