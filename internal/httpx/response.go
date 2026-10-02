package httpx

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		return
	}
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(
		w,
		status,
		map[string]string{
			"error": message,
		},
	)
}
