package event

import (
	"encoding/json"
	"fmt"
	"time"

	"doctor-service/internal/model"
	"doctor-service/internal/usecase"

	"github.com/nats-io/nats.go"
)

const doctorCreatedSubject = "doctors.created"

type doctorCreatedEvent struct {
	EventType      string `json:"event_type"`
	OccurredAt     string `json:"occurred_at"`
	ID             string `json:"id"`
	FullName       string `json:"full_name"`
	Specialization string `json:"specialization"`
	Email          string `json:"email"`
}

type NATSPublisher struct {
	nc *nats.Conn
}

func NewNATSPublisher(url string) (*NATSPublisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats %s: %w", url, err)
	}
	return &NATSPublisher{nc: nc}, nil
}

func (p *NATSPublisher) PublishDoctorCreated(doctor *model.Doctor) error {
	payload := doctorCreatedEvent{
		EventType:      doctorCreatedSubject,
		OccurredAt:     time.Now().UTC().Format(time.RFC3339),
		ID:             doctor.ID,
		FullName:       doctor.FullName,
		Specialization: doctor.Specialization,
		Email:          doctor.Email,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal doctor event: %w", err)
	}
	if err := p.nc.Publish(doctorCreatedSubject, body); err != nil {
		return fmt.Errorf("failed to publish %s: %w", doctorCreatedSubject, err)
	}
	return nil
}

func (p *NATSPublisher) Close() error {
	if p == nil || p.nc == nil {
		return nil
	}
	return p.nc.Drain()
}

var _ usecase.EventPublisher = (*NATSPublisher)(nil)

type NoopPublisher struct {
	reason error
}

func NewNoopPublisher(reason error) *NoopPublisher {
	return &NoopPublisher{reason: reason}
}

func (p *NoopPublisher) PublishDoctorCreated(_ *model.Doctor) error {
	return fmt.Errorf("doctor event publisher unavailable: %w", p.reason)
}

var _ usecase.EventPublisher = (*NoopPublisher)(nil)
