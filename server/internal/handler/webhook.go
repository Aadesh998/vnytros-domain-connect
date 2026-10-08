package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

func ReceiveWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("webhook receiver: read body: %v", err)
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("──── webhook received @ %s ────", time.Now().Format(time.RFC3339))
	log.Printf("  method:      %s", r.Method)
	log.Printf("  path:        %s", r.URL.Path)
	log.Printf("  remote:      %s", r.RemoteAddr)
	log.Printf("  user-agent:  %s", r.Header.Get("User-Agent"))
	log.Printf("  event:       %s", r.Header.Get("X-Vnytros-Event"))
	log.Printf("  content-type:%s", r.Header.Get("Content-Type"))

	var pretty map[string]any
	if json.Unmarshal(body, &pretty) == nil {
		out, _ := json.MarshalIndent(pretty, "  ", "  ")
		log.Printf("  body:\n  %s", out)
	} else {
		log.Printf("  body (raw): %s", body)
	}
	log.Printf("──── end webhook ────")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "received",
		"message": "webhook accepted",
	})
}
