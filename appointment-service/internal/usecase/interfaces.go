package usecase

import (
	"context"

	"appointment-service/internal/model"
)

type AppointmentRepository interface {
	Save(a *model.Appointment) error
	GetByID(id string) (*model.Appointment, error)
	GetAll() ([]*model.Appointment, error)
	Update(a *model.Appointment) error
}

type DoctorServiceClient interface {
	DoctorExists(ctx context.Context, doctorID string) (bool, error)
}

type CacheRepository interface {
	GetAppointment(id string) (*model.Appointment, bool, error)
	SetAppointment(a *model.Appointment) error
	GetAppointmentsList() ([]*model.Appointment, bool, error)
	SetAppointmentsList(appointments []*model.Appointment) error
	DeleteAppointmentsList() error
}

type EventPublisher interface {
	PublishAppointmentCreated(a *model.Appointment) error
	PublishAppointmentStatusUpdated(id, doctorID string, oldStatus, newStatus model.Status) error
}

type AppointmentUseCase interface {
	Create(ctx context.Context, title, description, doctorID string) (*model.Appointment, error)
	GetByID(id string) (*model.Appointment, error)
	GetAll() ([]*model.Appointment, error)
	UpdateStatus(ctx context.Context, id string, newStatus model.Status) (*model.Appointment, error)
}
