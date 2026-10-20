package dto

import (
	"bytes"
	"encoding/json"
)

type Optional[T any] struct {
	Value *T
	Set   bool
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.Value = nil
		return nil
	}

	var value T

	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	o.Value = &value

	return nil
}
