package jobqueue

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"notification-service/internal/logger"

	"github.com/redis/go-redis/v9"
)

const (
	idempotencyPrefix = "idempotency:"
	doneValue         = "done"
	doneTTL           = 24 * time.Hour
	maxRetries        = 3
)

type Job struct {
	IdempotencyKey string `json:"idempotency_key"`
	AppointmentID  string `json:"appointment_id"`
	DoctorID       string `json:"doctor_id"`
	OccurredAt     string `json:"occurred_at"`
	Channel        string `json:"channel"`
	Recipient      string `json:"recipient"`
	Message        string `json:"message"`
}

type Queue struct {
	redisClient *redis.Client
	logger      *logger.Logger
	gatewayURL  string
	httpClient  *http.Client

	jobs chan Job
	wg   sync.WaitGroup
}

func New(redisClient *redis.Client, lg *logger.Logger, gatewayURL string, workerPoolSize int) *Queue {
	if workerPoolSize <= 0 {
		workerPoolSize = 3
	}

	q := &Queue{
		redisClient: redisClient,
		logger:      lg,
		gatewayURL:  strings.TrimSuffix(gatewayURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		jobs: make(chan Job, workerPoolSize*10),
	}

	for i := 0; i < workerPoolSize; i++ {
		q.wg.Add(1)
		go q.worker()
	}

	return q
}

func (q *Queue) Enqueue(job Job) error {
	if job.IdempotencyKey == "" {
		return fmt.Errorf("idempotency_key is required")
	}

	done, err := q.isDone(job.IdempotencyKey)
	if err != nil {
		q.logger.Job("warn", job.IdempotencyKey, 1, "retry", fmt.Errorf("idempotency check failed: %w", err))
	} else if done {
		q.logger.Job("info", job.IdempotencyKey, 1, "duplicate", nil)
		return nil
	}

	q.jobs <- job
	q.logger.Job("info", job.IdempotencyKey, 1, "enqueued", nil)

	return nil
}

func (q *Queue) Shutdown() {
	close(q.jobs)
	q.wg.Wait()
}

func (q *Queue) worker() {
	defer q.wg.Done()

	for job := range q.jobs {
		q.process(job)
	}
}

func (q *Queue) process(job Job) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		q.logger.Job("info", job.IdempotencyKey, attempt, "processing", nil)

		err := q.callGateway(job)
		if err == nil {
			if markErr := q.markDone(job.IdempotencyKey); markErr != nil {
				err = fmt.Errorf("failed to mark idempotency key as done: %w", markErr)
			}
		}

		if err == nil {
			q.logger.Job("info", job.IdempotencyKey, attempt, "success", nil)
			return
		}

		if attempt < maxRetries {
			q.logger.Job("warn", job.IdempotencyKey, attempt, "retry", err)
			time.Sleep(backoffDuration(attempt))
			continue
		}

		q.logger.Job("error", job.IdempotencyKey, attempt, "dead_letter", err)
	}
}

func (q *Queue) callGateway(job Job) error {
	payload, err := json.Marshal(map[string]any{
		"idempotency_key": job.IdempotencyKey,
		"channel":         job.Channel,
		"recipient":       job.Recipient,
		"message":         job.Message,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal gateway payload: %w", err)
	}

	resp, err := q.httpClient.Post(q.gatewayURL+"/notify", "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("gateway request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusServiceUnavailable {
		return fmt.Errorf("gateway returned 503")
	}

	return fmt.Errorf("gateway returned unexpected status: %d", resp.StatusCode)
}

func (q *Queue) isDone(idempotencyKey string) (bool, error) {
	value, err := q.redisClient.Get(context.Background(), redisKey(idempotencyKey)).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	return value == doneValue, nil
}

func (q *Queue) markDone(idempotencyKey string) error {
	return q.redisClient.Set(context.Background(), redisKey(idempotencyKey), doneValue, doneTTL).Err()
}

func backoffDuration(attempt int) time.Duration {
	switch attempt {
	case 1:
		return time.Second
	case 2:
		return 2 * time.Second
	default:
		return 4 * time.Second
	}
}

func BuildIdempotencyKey(eventType, appointmentID, occurredAt string) string {
	sum := sha256.Sum256([]byte(eventType + appointmentID + occurredAt))
	return hex.EncodeToString(sum[:])
}

func redisKey(idempotencyKey string) string {
	return idempotencyPrefix + idempotencyKey
}
