package usecase

import (
	"errors"
	"log"

	"doctor-service/internal/model"

	"github.com/google/uuid"
)

type doctorUsecase struct {
	repo      DoctorRepository
	cache     CacheRepository
	publisher EventPublisher
}

func New(repo DoctorRepository, cache CacheRepository, publisher EventPublisher) DoctorUsecase {
	return &doctorUsecase{
		repo:      repo,
		cache:     cache,
		publisher: publisher,
	}
}

func (us *doctorUsecase) Create(fullName, specialization, email string) (*model.Doctor, error) {
	if fullName == "" {
		return nil, errors.New("full_name is required")
	}
	if email == "" {
		return nil, errors.New("email is required")
	}

	exists, err := us.repo.EmailExists(email)
	if err != nil {
		log.Printf("[ERROR] failed to check email uniqueness for %s: %v", email, err)
		return nil, err
	}
	if exists {
		log.Printf("[WARN] attempt to register duplicate email: %s", email)
		return nil, errors.New("email already in use")
	}

	doctor := &model.Doctor{
		ID:             uuid.NewString(),
		FullName:       fullName,
		Specialization: specialization,
		Email:          email,
	}

	if err := us.repo.Save(doctor); err != nil {
		log.Printf("[ERROR] failed to save doctor %s: %v", doctor.ID, err)
		return nil, err
	}

	if err := us.cache.DeleteDoctorsList(); err != nil {
		log.Printf("[ERROR] failed to invalidate doctors list cache after create: %v", err)
	}

	if err := us.publisher.PublishDoctorCreated(doctor); err != nil {
		log.Printf("[ERROR] failed to publish doctors.created event for doctor %s: %v", doctor.ID, err)
	}

	log.Printf("[INFO] doctor created: id=%s email=%s", doctor.ID, doctor.Email)
	return doctor, nil
}

func (uc *doctorUsecase) GetByID(id string) (*model.Doctor, error) {
	if cachedDoctor, hit, err := uc.cache.GetDoctor(id); err != nil {
		log.Printf("[ERROR] failed to read doctor %s from cache: %v", id, err)
	} else if hit {
		return cachedDoctor, nil
	}

	doctor, err := uc.repo.GetByID(id)
	if err != nil {
		log.Printf("[WARN] doctor not found: id=%s", id)
		return nil, err
	}

	if err := uc.cache.SetDoctor(doctor); err != nil {
		log.Printf("[ERROR] failed to write doctor %s to cache: %v", id, err)
	}

	return doctor, nil
}

func (uc *doctorUsecase) GetAll() ([]*model.Doctor, error) {
	if cachedDoctors, hit, err := uc.cache.GetDoctorsList(); err != nil {
		log.Printf("[ERROR] failed to read doctors list from cache: %v", err)
	} else if hit {
		return cachedDoctors, nil
	}

	doctors, err := uc.repo.GetAll()
	if err != nil {
		log.Printf("[ERROR] failed to retrieve all doctors: %v", err)
		return nil, err
	}

	if err := uc.cache.SetDoctorsList(doctors); err != nil {
		log.Printf("[ERROR] failed to write doctors list to cache: %v", err)
	}

	log.Printf("[INFO] retrieved %d doctors", len(doctors))
	return doctors, nil
}
