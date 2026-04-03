package usecase

import (
	"errors"
	"log"

	"doctor-service/internal/model"

	"github.com/google/uuid"
)

type doctorUsecase struct {
	repo DoctorRepository
}

func New(repo DoctorRepository) DoctorUsecase {
	return &doctorUsecase{
		repo: repo,
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

	log.Printf("[INFO] doctor created: id=%s email=%s", doctor.ID, doctor.Email)
	return doctor, nil
}

func (uc *doctorUsecase) GetByID(id string) (*model.Doctor, error) {
	doctor, err := uc.repo.GetByID(id)
	if err != nil {
		log.Printf("[WARN] doctor not found: id=%s", id)
		return nil, err
	}
	return doctor, nil
}

func (uc *doctorUsecase) GetAll() ([]*model.Doctor, error) {
	doctors, err := uc.repo.GetAll()
	if err != nil {
		log.Printf("[ERROR] failed to retrieve all doctors: %v", err)
		return nil, err
	}
	log.Printf("[INFO] retrieved %d doctors", len(doctors))
	return doctors, nil
}
