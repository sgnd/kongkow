package store

import (
	"encoding/json"

	"github.com/sgnd/kongkow/internal/sealrow"
)

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

func newID() (string, error) { return sealrow.NewID() }
