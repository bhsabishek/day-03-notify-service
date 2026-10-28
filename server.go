package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

type Server struct {
	apiKey string
	source string

	mu     sync.Mutex
	nextID int
}

func NewServer(apiKey, source string) *Server {
	return &Server{apiKey: apiKey, source: source}
}

type notifyRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /config/check", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"api_key_loaded": s.apiKey != "",
			"source":         s.source,
		})
	})

	mux.HandleFunc("POST /notify", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		var req notifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		if strings.TrimSpace(req.To) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "to is required"})
			return
		}

		if strings.TrimSpace(req.Message) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
			return
		}

		s.mu.Lock()
		s.nextID++
		id := s.nextID
		s.mu.Unlock()

		writeJSON(w, http.StatusAccepted, map[string]any{
			"id":        id,
			"status":    "queued",
			"key_last4": last4(s.apiKey),
		})
	})

	return mux
}

// last4 returns the final four characters of a secret, never more than that.
func last4(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return key[len(key)-4:]
}
