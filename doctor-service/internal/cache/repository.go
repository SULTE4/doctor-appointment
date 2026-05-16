package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"doctor-service/internal/model"
	"doctor-service/internal/usecase"

	"github.com/redis/go-redis/v9"
)

const (
	doctorKeyPrefix = "doctor:"
	doctorsListKey  = "doctors:list"
	defaultTTL      = 60 * time.Second
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

func (c *RedisCache) GetDoctor(id string) (*model.Doctor, bool, error) {
	value, err := c.client.Get(context.Background(), doctorKey(id)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var doctor model.Doctor
	if err := json.Unmarshal([]byte(value), &doctor); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal doctor cache entry: %w", err)
	}

	return &doctor, true, nil
}

func (c *RedisCache) SetDoctor(doctor *model.Doctor) error {
	payload, err := json.Marshal(doctor)
	if err != nil {
		return fmt.Errorf("failed to marshal doctor cache entry: %w", err)
	}

	return c.client.Set(context.Background(), doctorKey(doctor.ID), payload, c.ttl).Err()
}

func (c *RedisCache) GetDoctorsList() ([]*model.Doctor, bool, error) {
	value, err := c.client.Get(context.Background(), doctorsListKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var doctors []*model.Doctor
	if err := json.Unmarshal([]byte(value), &doctors); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal doctors list cache entry: %w", err)
	}

	return doctors, true, nil
}

func (c *RedisCache) SetDoctorsList(doctors []*model.Doctor) error {
	payload, err := json.Marshal(doctors)
	if err != nil {
		return fmt.Errorf("failed to marshal doctors list cache entry: %w", err)
	}

	return c.client.Set(context.Background(), doctorsListKey, payload, c.ttl).Err()
}

func (c *RedisCache) DeleteDoctorsList() error {
	return c.client.Del(context.Background(), doctorsListKey).Err()
}

func doctorKey(id string) string {
	return doctorKeyPrefix + id
}

var _ usecase.CacheRepository = (*RedisCache)(nil)

type NoopCache struct{}

func NewNoop() *NoopCache {
	return &NoopCache{}
}

func (c *NoopCache) GetDoctor(_ string) (*model.Doctor, bool, error) {
	return nil, false, nil
}

func (c *NoopCache) SetDoctor(_ *model.Doctor) error {
	return nil
}

func (c *NoopCache) GetDoctorsList() ([]*model.Doctor, bool, error) {
	return nil, false, nil
}

func (c *NoopCache) SetDoctorsList(_ []*model.Doctor) error {
	return nil
}

func (c *NoopCache) DeleteDoctorsList() error {
	return nil
}

var _ usecase.CacheRepository = (*NoopCache)(nil)
