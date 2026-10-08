package utils

import (
	"domain-connect-backend/internal/errorz"
	"encoding/json"
	"log"
	"net/http"
)

func SendSuccess(w http.ResponseWriter, resp any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}
}
