package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

type notifyRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	Channel        string `json:"channel"`
	Recipient      string `json:"recipient"`
	Message        string `json:"message"`
}

type gateway struct {
	mu   sync.Mutex
	seen map[string]struct{}
	rnd  *rand.Rand
}

func newGateway() *gateway {
	return &gateway{
		seen: make(map[string]struct{}),
		rnd:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (g *gateway) notify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req notifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	g.logRequest(req)

	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.seen[req.IdempotencyKey]; exists {
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
		return
	}

	if g.rnd.Float64() < 0.2 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "temporary_unavailable"})
		return
	}

	g.seen[req.IdempotencyKey] = struct{}{}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (g *gateway) logRequest(req notifyRequest) {
	entry := map[string]any{
		"time":            time.Now().UTC().Format(time.RFC3339),
		"idempotency_key": req.IdempotencyKey,
		"channel":         req.Channel,
		"recipient":       req.Recipient,
		"message":         req.Message,
	}
	if err := json.NewEncoder(os.Stdout).Encode(entry); err != nil {
		log.Printf("failed to write gateway log: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}

func main() {
	port := os.Getenv("GATEWAY_PORT")
	if port == "" {
		port = "8090"
	}

	g := newGateway()

	mux := http.NewServeMux()
	mux.HandleFunc("/notify", g.notify)

	addr := ":" + port
	log.Printf("mock gateway listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("mock gateway failed: %v", err)
	}
}
