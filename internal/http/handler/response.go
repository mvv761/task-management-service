package handler

import (
	"encoding/json"
	"net/http"
)

func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(body)
	return err
}
