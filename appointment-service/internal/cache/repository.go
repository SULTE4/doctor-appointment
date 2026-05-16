package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"

	"github.com/redis/go-redis/v9"
)

const (
	appointmentKeyPrefix = "appointment:"
	appointmentsListKey  = "appointments:list"
	defaultTTL           = 60 * time.Second
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedis(client *redis.Client, ttlSeconds int) *RedisCache {
	ttl := defaultTTL
	if ttlSeconds > 0 {
		ttl = time.Duration(ttlSeconds) * time.Second
	}

	return &RedisCache{
		client: client,
		ttl:    ttl,
	}
}

func (c *RedisCache) GetAppointment(id string) (*model.Appointment, bool, error) {
	value, err := c.client.Get(context.Background(), appointmentKey(id)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var appointment model.Appointment
	if err := json.Unmarshal([]byte(value), &appointment); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal appointment cache entry: %w", err)
	}

	return &appointment, true, nil
}

func (c *RedisCache) SetAppointment(a *model.Appointment) error {
	payload, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("failed to marshal appointment cache entry: %w", err)
	}

	return c.client.Set(context.Background(), appointmentKey(a.ID), payload, c.ttl).Err()
}

func (c *RedisCache) GetAppointmentsList() ([]*model.Appointment, bool, error) {
	value, err := c.client.Get(context.Background(), appointmentsListKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var appointments []*model.Appointment
	if err := json.Unmarshal([]byte(value), &appointments); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal appointments list cache entry: %w", err)
	}

	return appointments, true, nil
}

func (c *RedisCache) SetAppointmentsList(appointments []*model.Appointment) error {
	payload, err := json.Marshal(appointments)
	if err != nil {
		return fmt.Errorf("failed to marshal appointments list cache entry: %w", err)
	}

	return c.client.Set(context.Background(), appointmentsListKey, payload, c.ttl).Err()
}

func (c *RedisCache) DeleteAppointmentsList() error {
	return c.client.Del(context.Background(), appointmentsListKey).Err()
}

func appointmentKey(id string) string {
	return appointmentKeyPrefix + id
}

var _ usecase.CacheRepository = (*RedisCache)(nil)

type NoopCache struct{}

func NewNoop() *NoopCache {
	return &NoopCache{}
}

func (c *NoopCache) GetAppointment(_ string) (*model.Appointment, bool, error) {
	return nil, false, nil
}

func (c *NoopCache) SetAppointment(_ *model.Appointment) error {
	return nil
}

func (c *NoopCache) GetAppointmentsList() ([]*model.Appointment, bool, error) {
	return nil, false, nil
}

func (c *NoopCache) SetAppointmentsList(_ []*model.Appointment) error {
	return nil
}

func (c *NoopCache) DeleteAppointmentsList() error {
	return nil
}

var _ usecase.CacheRepository = (*NoopCache)(nil)
