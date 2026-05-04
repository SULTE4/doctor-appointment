package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"appointment-service/internal/model"

	"github.com/google/uuid"
)

type appointmentUseCase struct {
	repo         AppointmentRepository
	doctorClient DoctorServiceClient
	publisher    EventPublisher
}

func New(repo AppointmentRepository, dc DoctorServiceClient, publisher EventPublisher) AppointmentUseCase {
	return &appointmentUseCase{
		repo:         repo,
		doctorClient: dc,
		publisher:    publisher,
	}
}

func (uc *appointmentUseCase) Create(ctx context.Context, title, description, doctorID string) (*model.Appointment, error) {
	if title == "" {
		return nil, errors.New("title is required")
	}
	if doctorID == "" {
		return nil, errors.New("doctor_id is required")
	}

	exists, err := uc.doctorClient.DoctorExists(ctx, doctorID)
	if err != nil {
		log.Printf("[ERROR] Doctor Service unavailable when creating appointment for doctor %s: %v", doctorID, err)
		return nil, fmt.Errorf("SERVICE_UNAVAILABLE: %w", err)
	}
	if !exists {
		log.Printf("[WARN] appointment creation rejected: doctor %s not found", doctorID)
		return nil, errors.New("DOCTOR_NOT_FOUND: doctor does not exist")
	}

	now := time.Now()
	a := &model.Appointment{
		ID:          uuid.NewString(),
		Title:       title,
		Description: description,
		DoctorID:    doctorID,
		Status:      model.StatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.repo.Save(a); err != nil {
		log.Printf("[ERROR] failed to save appointment %s: %v", a.ID, err)
		return nil, err
	}

	if err := uc.publisher.PublishAppointmentCreated(a); err != nil {
		log.Printf("[ERROR] failed to publish appointments.created event for appointment %s: %v", a.ID, err)
	}

	log.Printf("[INFO] appointment created: id=%s doctor_id=%s", a.ID, a.DoctorID)
	return a, nil
}

func (uc *appointmentUseCase) GetByID(id string) (*model.Appointment, error) {
	a, err := uc.repo.GetByID(id)
	if err != nil {
		log.Printf("[WARN] appointment not found: id=%s", id)
		return nil, err
	}
	return a, nil
}

func (uc *appointmentUseCase) GetAll() ([]*model.Appointment, error) {
	appointments, err := uc.repo.GetAll()
	if err != nil {
		log.Printf("[ERROR] failed to retrieve all appointments: %v", err)
		return nil, err
	}
	log.Printf("[INFO] retrieved %d appointments", len(appointments))
	return appointments, nil
}

func (uc *appointmentUseCase) UpdateStatus(ctx context.Context, id string, newStatus model.Status) (*model.Appointment, error) {
	if !newStatus.IsValid() {
		return nil, errors.New("status must be one of: new, in_progress, done")
	}

	a, err := uc.repo.GetByID(id)
	if err != nil {
		log.Printf("[WARN] appointment not found for status update: id=%s", id)
		return nil, err
	}

	if a.Status == model.StatusDone && newStatus == model.StatusNew {
		log.Printf("[WARN] forbidden status transition for appointment %s: done -> new", id)
		return nil, errors.New("FORBIDDEN_TRANSITION: cannot move from done back to new")
	}

	exists, err := uc.doctorClient.DoctorExists(ctx, a.DoctorID)
	if err != nil {
		log.Printf("[ERROR] Doctor Service unavailable when updating appointment %s for doctor %s: %v", id, a.DoctorID, err)
		return nil, fmt.Errorf("SERVICE_UNAVAILABLE: %w", err)
	}
	if !exists {
		log.Printf("[WARN] appointment status update rejected: doctor %s not found", a.DoctorID)
		return nil, errors.New("DOCTOR_NOT_FOUND: doctor does not exist")
	}

	oldStatus := a.Status
	a.Status = newStatus
	a.UpdatedAt = time.Now()

	if err := uc.repo.Update(a); err != nil {
		log.Printf("[ERROR] failed to update appointment %s: %v", id, err)
		return nil, err
	}

	if err := uc.publisher.PublishAppointmentStatusUpdated(a.ID, oldStatus, newStatus); err != nil {
		log.Printf("[ERROR] failed to publish appointments.status_updated event for appointment %s: %v", a.ID, err)
	}

	log.Printf("[INFO] appointment %s status updated to %s", id, newStatus)
	return a, nil
}
