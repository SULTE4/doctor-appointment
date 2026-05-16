package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"doctor-service/internal/cache"
	"doctor-service/internal/event"
	"doctor-service/internal/middleware"
	"doctor-service/internal/repository"
	grpcHandler "doctor-service/internal/transport/grpc"
	"doctor-service/internal/usecase"
	doctorpb "doctor-service/proto"

	_ "github.com/lib/pq"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func Run() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("failed load .env data")
	}
	port := os.Getenv("DOCTOR_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}
	dsn := os.Getenv("DB_DSN")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("failed to init db, error: ", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping database, error: ", err)
	}

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Fatal("failed at init migrate, error:", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("no change at migration")
		} else {
			log.Fatal("failed migrate up, error:", err)
		}
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	cacheTTLSeconds := readIntEnv("CACHE_TTL_SECONDS", 60)
	rateLimitRPM := readIntEnv("RATE_LIMIT_RPM", 100)

	var redisClient *redis.Client
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("[WARN] failed to parse REDIS_URL (%s): %v", redisURL, err)
	} else {
		redisClient = redis.NewClient(redisOpts)

		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if pingErr := redisClient.Ping(pingCtx).Err(); pingErr != nil {
			log.Printf("[WARN] redis unavailable at startup, continuing in degraded mode: %v", pingErr)
			_ = redisClient.Close()
			redisClient = nil
		}
	}
	if redisClient != nil {
		defer func() {
			if closeErr := redisClient.Close(); closeErr != nil {
				log.Printf("[WARN] failed to close redis client: %v", closeErr)
			}
		}()
	}

	publisher := usecase.EventPublisher(event.NewNoopPublisher(errors.New("event publisher is not configured")))
	natsPublisher, err := event.NewNATSPublisher(natsURL)
	if err != nil {
		log.Printf("[WARN] broker unavailable at startup, service continues with best-effort publishing disabled: %v", err)
		publisher = event.NewNoopPublisher(err)
	} else {
		publisher = natsPublisher
		defer func() {
			if closeErr := natsPublisher.Close(); closeErr != nil {
				log.Printf("[WARN] failed to close NATS publisher: %v", closeErr)
			}
		}()
	}

	repo := repository.New(db)
	cacheRepo := usecase.CacheRepository(cache.NewNoop())
	if redisClient != nil {
		cacheRepo = cache.NewRedis(redisClient, cacheTTLSeconds)
	}

	uc := usecase.New(repo, cacheRepo, publisher)
	h := grpcHandler.NewHandler(uc)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("[FATAL] failed to listen on :%s: %v", port, err)
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(middleware.NewRedisRateLimiterInterceptor(redisClient, rateLimitRPM)))
	doctorpb.RegisterDoctorServiceServer(server, h)

	log.Printf("[INFO] Doctor Service gRPC listening on :%s", port)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("[FATAL] failed to serve doctor gRPC server: %v", err)
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
