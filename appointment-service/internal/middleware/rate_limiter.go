package middleware

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	defaultRPM = 100
	windowSize = time.Minute
)

func NewRedisRateLimiterInterceptor(client *redis.Client, rpm int) grpc.UnaryServerInterceptor {
	if rpm <= 0 {
		rpm = defaultRPM
	}

	if client == nil {
		return passthroughInterceptor
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		clientID := clientIPFromContext(ctx)
		allowed, retryAfter, err := allowRequest(ctx, client, clientID, rpm)
		if err != nil {
			log.Printf("[WARN] rate limiter error on %s: %v", info.FullMethod, err)
			return handler(ctx, req)
		}
		if !allowed {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded, retry after %ds", int(retryAfter.Seconds()))
		}

		return handler(ctx, req)
	}
}

func allowRequest(ctx context.Context, client *redis.Client, clientID string, rpm int) (bool, time.Duration, error) {
	now := time.Now().UTC()
	nowMs := now.UnixMilli()
	windowStartMs := now.Add(-windowSize).UnixMilli()
	key := fmt.Sprintf("ratelimit:%s", clientID)
	member := strconv.FormatInt(now.UnixNano(), 10)

	pipe := client.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(windowStartMs, 10))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(nowMs),
		Member: member,
	})
	pipe.Expire(ctx, key, windowSize+10*time.Second)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, 0, err
	}

	count := countCmd.Val()
	if count >= int64(rpm) {
		_, _ = client.ZRem(ctx, key, member).Result()

		retryAfter := time.Second
		oldest, err := client.ZRangeWithScores(ctx, key, 0, 0).Result()
		if err == nil && len(oldest) > 0 {
			oldestMs := int64(oldest[0].Score)
			elapsed := nowMs - oldestMs
			remainingMs := int64(windowSize/time.Millisecond) - elapsed
			if remainingMs < 1000 {
				remainingMs = 1000
			}
			retryAfter = time.Duration(remainingMs) * time.Millisecond
		}

		return false, retryAfter, nil
	}

	return true, 0, nil
}

func clientIPFromContext(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return "unknown"
	}

	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return p.Addr.String()
	}

	return host
}

func passthroughInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	return handler(ctx, req)
}
