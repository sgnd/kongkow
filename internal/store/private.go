package store

import "os"

func MkdirPrivate(parts ...string) error {
	if err := ensureHome(); err != nil {
		return err
	}
	return os.MkdirAll(Path(parts...), 0o700)
}
