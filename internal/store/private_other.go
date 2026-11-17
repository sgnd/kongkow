//go:build !windows

package store

import "os"

func privateHome(string) error { return nil }

func tightenHome(h string) {
	fi, err := os.Lstat(h)
	if err != nil || !fi.IsDir() || fi.Mode().Perm()&0o077 == 0 {
		return
	}
	for _, f := range []string{StoreFile, "store.key"} {
		if _, err := os.Lstat(Path(f)); err == nil {
			_ = os.Chmod(h, 0o700)
			return
		}
	}
}
