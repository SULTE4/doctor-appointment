package usecase

import "appointment-service/internal/model"

type AppointmentRepository interface {
	Save(a *model.Appointment) error
	GetByID(id string) (*model.Appointment, error)
	GetAll() ([]*model.Appointment, error)
	Update(a *model.Appointment) error
}

type DoctorServiceClient interface {
	DoctorExists(doctorID string) (bool, error)
}

type AppointmentUseCase interface {
	Create(title, description, doctorID string) (*model.Appointment, error)
	GetByID(id string) (*model.Appointment, error)
	GetAll() ([]*model.Appointment, error)
	UpdateStatus(id string, newStatus model.Status) (*model.Appointment, error)
}
