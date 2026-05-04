package subscriber

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type Subscriber struct {
	nc   *nats.Conn
	subs []*nats.Subscription

	mu  sync.Mutex
	enc *json.Encoder
}

func ConnectWithRetry(url string, maxAttempts int, initialBackoff time.Duration) (*nats.Conn, error) {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	backoff := initialBackoff
	if backoff <= 0 {
		backoff = time.Second
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		nc, err := nats.Connect(url, nats.Name("notification-service"))
		if err == nil {
			log.Printf("[INFO] connected to broker at %s", url)
			return nc, nil
		}

		lastErr = err
		if attempt == maxAttempts {
			break
		}

		log.Printf("[WARN] broker unavailable (attempt %d/%d): %v; retrying in %s", attempt, maxAttempts, err, backoff)
		time.Sleep(backoff)
		backoff *= 2
	}

	return nil, fmt.Errorf("failed to connect to broker at %s after %d attempts: %w", url, maxAttempts, lastErr)
}

func New(nc *nats.Conn) (*Subscriber, error) {
	if nc == nil {
		return nil, fmt.Errorf("nats connection is nil")
	}

	return &Subscriber{
		nc:  nc,
		enc: json.NewEncoder(os.Stdout),
	}, nil
}

func (s *Subscriber) Subscribe(subjects []string) error {
	for _, subject := range subjects {
		sub, err := s.nc.Subscribe(subject, s.handleMessage)
		if err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
		}
		s.subs = append(s.subs, sub)
	}

	s.nc.Flush()
	if err := s.nc.LastError(); err != nil {
		return fmt.Errorf("broker flush failed: %w", err)
	}

	log.Printf("[INFO] subscribed to %d subjects", len(subjects))
	return nil
}

func (s *Subscriber) Drain() error {
	if s == nil || s.nc == nil {
		return nil
	}
	return s.nc.Drain()
}

func (s *Subscriber) handleMessage(msg *nats.Msg) {
	var event map[string]any
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		s.writeLog(map[string]any{
			"time":    time.Now().UTC().Format(time.RFC3339),
			"subject": msg.Subject,
			"error":   fmt.Sprintf("failed to deserialize payload: %v", err),
			"raw":     string(msg.Data),
		})
		return
	}

	s.writeLog(event)
}

func (s *Subscriber) writeLog(entry map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.enc.Encode(entry); err != nil {
		log.Printf("[ERROR] failed to write log output: %v", err)
	}
}
