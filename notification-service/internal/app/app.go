package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"notification-service/internal/jobqueue"
	"notification-service/internal/logger"
	"notification-service/internal/subscriber"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func Run() error {
	_ = godotenv.Load()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8090"
	}
	workerPoolSize := readIntEnv("WORKER_POOL_SIZE", 3)

	nc, err := subscriber.ConnectWithRetry(natsURL, 5, time.Second)
	if err != nil {
		return err
	}
	defer nc.Close()

	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("invalid REDIS_URL %s: %w", redisURL, err)
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer pingCancel()
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis at %s: %w", redisURL, err)
	}

	lg := logger.New()
	queue := jobqueue.New(redisClient, lg, gatewayURL, workerPoolSize)
	defer queue.Shutdown()

	s, err := subscriber.New(nc, func(subject string, payload []byte) {
		handleMessage(lg, queue, subject, payload)
	})
	if err != nil {
		return err
	}

	subjects := []string{
		"doctors.created",
		"appointments.created",
		"appointments.status_updated",
	}
	if err := s.Subscribe(subjects); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	return s.Drain()
}

func handleMessage(lg *logger.Logger, queue *jobqueue.Queue, subject string, payload []byte) {
	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("[ERROR] failed to deserialize message on %s: %v", subject, err)
		return
	}

	lg.Event(subject, event)

	if subject != "appointments.status_updated" {
		return
	}

	newStatus, _ := event["new_status"].(string)
	if newStatus != "done" {
		return
	}

	eventType, _ := event["event_type"].(string)
	appointmentID, _ := event["id"].(string)
	occurredAt, _ := event["occurred_at"].(string)
	doctorID, _ := event["doctor_id"].(string)
	if eventType == "" || appointmentID == "" || occurredAt == "" || doctorID == "" {
		log.Printf("[WARN] missing required fields for done status job payload: event_type=%q id=%q occurred_at=%q doctor_id=%q", eventType, appointmentID, occurredAt, doctorID)
		return
	}

	idempotencyKey := jobqueue.BuildIdempotencyKey(eventType, appointmentID, occurredAt)
	job := jobqueue.Job{
		IdempotencyKey: idempotencyKey,
		AppointmentID:  appointmentID,
		DoctorID:       doctorID,
		OccurredAt:     occurredAt,
		Channel:        "email",
		Recipient:      "patient@clinic.kz",
		Message:        fmt.Sprintf("Your appointment %s with doctor %s is complete.", appointmentID, doctorID),
	}

	if err := queue.Enqueue(job); err != nil {
		lg.Job("error", idempotencyKey, 1, "dead_letter", err)
	}
}

func readIntEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("[WARN] invalid %s value %q, fallback to %d", key, raw, fallback)
		return fallback
	}

	return value
}
