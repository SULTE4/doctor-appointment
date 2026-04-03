package repository

import (
	"errors"
	"sync"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"
)

type appointmentRepo struct {
	appointments map[string]*model.Appointment
	mu           sync.RWMutex
}

func New() usecase.AppointmentRepository {
	return &appointmentRepo{
		appointments: make(map[string]*model.Appointment),
	}
}

func (r *appointmentRepo) Save(a *model.Appointment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.appointments[a.ID] = a
	return nil
}

func (r *appointmentRepo) GetByID(id string) (*model.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if a, ok := r.appointments[id]; ok {
		return a, nil
	}
	return nil, errors.New("appointment not found")
}

func (r *appointmentRepo) GetAll() ([]*model.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*model.Appointment, 0, len(r.appointments))
	for _, a := range r.appointments {
		result = append(result, a)
	}
	return result, nil
}

func (r *appointmentRepo) Update(a *model.Appointment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.appointments[a.ID]; !ok {
		return errors.New("appointment not found")
	}
	r.appointments[a.ID] = a
	return nil
}
