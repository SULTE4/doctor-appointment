package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notification-service/internal/subscriber"

	"github.com/joho/godotenv"
)

func Run() error {
	_ = godotenv.Load()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	nc, err := subscriber.ConnectWithRetry(natsURL, 5, time.Second)
	if err != nil {
		return err
	}

	s, err := subscriber.New(nc)
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
