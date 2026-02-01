package helpers

import (
	"encoding/json"
	"log"
	"net/http"
)

func ErrorResponse(w http.ResponseWriter, statusError int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusError)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   message,
		"details": err.Error(),
	})
	log.Println(err)
}
