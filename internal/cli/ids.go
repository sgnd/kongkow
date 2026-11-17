package cli

import "github.com/sgnd/kongkow/internal/sealrow"

func storeNewID() (string, error) { return sealrow.NewID() }
