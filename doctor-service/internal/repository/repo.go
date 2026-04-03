package repository

import (
	"errors"
	"sync"

	"doctor-service/internal/model"
	"doctor-service/internal/usecase"
)

type DoctorRepo struct {
	doctors map[string]*model.Doctor
	mu      sync.RWMutex
}

func New() usecase.DoctorRepository {
	return &DoctorRepo{
		doctors: make(map[string]*model.Doctor),
	}
}

func (r *DoctorRepo) Save(doctor *model.Doctor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.doctors[doctor.ID] = doctor

	return nil
}

func (r *DoctorRepo) GetAll() ([]*model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	doctors := make([]*model.Doctor, 0, len(r.doctors))
	for _, doctor := range r.doctors {
		doctors = append(doctors, doctor)
	}

	return doctors, nil
}

func (r *DoctorRepo) GetByID(id string) (*model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	doctor, ok := r.doctors[id]
	if !ok {
		return nil, errors.New("doctor not found")
	}

	return doctor, nil
}

func (r *DoctorRepo) EmailExists(email string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, doctor := range r.doctors {
		if doctor.Email == email {
			return true, nil
		}
	}

	return false, nil
}
