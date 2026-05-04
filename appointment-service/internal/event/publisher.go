package event

import (
	"encoding/json"
	"fmt"
	"time"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"

	"github.com/nats-io/nats.go"
)

const (
	appointmentCreatedSubject       = "appointments.created"
	appointmentStatusUpdatedSubject = "appointments.status_updated"
)

type appointmentCreatedEvent struct {
	EventType  string `json:"event_type"`
	OccurredAt string `json:"occurred_at"`
	ID         string `json:"id"`
	Title      string `json:"title"`
	DoctorID   string `json:"doctor_id"`
	Status     string `json:"status"`
}

type appointmentStatusUpdatedEvent struct {
	EventType  string `json:"event_type"`
	OccurredAt string `json:"occurred_at"`
	ID         string `json:"id"`
	OldStatus  string `json:"old_status"`
	NewStatus  string `json:"new_status"`
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

func (p *NATSPublisher) PublishAppointmentCreated(a *model.Appointment) error {
	payload := appointmentCreatedEvent{
		EventType:  appointmentCreatedSubject,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		ID:         a.ID,
		Title:      a.Title,
		DoctorID:   a.DoctorID,
		Status:     string(a.Status),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal appointment created event: %w", err)
	}
	if err := p.nc.Publish(appointmentCreatedSubject, body); err != nil {
		return fmt.Errorf("failed to publish %s: %w", appointmentCreatedSubject, err)
	}
	return nil
}

func (p *NATSPublisher) PublishAppointmentStatusUpdated(id string, oldStatus, newStatus model.Status) error {
	payload := appointmentStatusUpdatedEvent{
		EventType:  appointmentStatusUpdatedSubject,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		ID:         id,
		OldStatus:  string(oldStatus),
		NewStatus:  string(newStatus),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal appointment status event: %w", err)
	}
	if err := p.nc.Publish(appointmentStatusUpdatedSubject, body); err != nil {
		return fmt.Errorf("failed to publish %s: %w", appointmentStatusUpdatedSubject, err)
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

func (p *NoopPublisher) PublishAppointmentCreated(_ *model.Appointment) error {
	return fmt.Errorf("appointment event publisher unavailable: %w", p.reason)
}

func (p *NoopPublisher) PublishAppointmentStatusUpdated(_ string, _ model.Status, _ model.Status) error {
	return fmt.Errorf("appointment event publisher unavailable: %w", p.reason)
}

var _ usecase.EventPublisher = (*NoopPublisher)(nil)
