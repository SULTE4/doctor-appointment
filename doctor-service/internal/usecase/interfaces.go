package usecase

import "doctor-service/internal/model"

type DoctorUsecase interface {
	Create(fullName, specialization, email string) (*model.Doctor, error)
	GetAll() ([]*model.Doctor, error)
	GetByID(id string) (*model.Doctor, error)
}

type EventPublisher interface {
	PublishDoctorCreated(doctor *model.Doctor) error
}

type DoctorRepository interface {
	Save(doctor *model.Doctor) error
	GetAll() ([]*model.Doctor, error)
	GetByID(id string) (*model.Doctor, error)
	EmailExists(email string) (bool, error)
}
