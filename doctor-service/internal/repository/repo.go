package repository

import (
	"database/sql"
	"errors"

	"doctor-service/internal/model"
	"doctor-service/internal/usecase"
)

type DoctorRepo struct {
	db *sql.DB
}

func New(db *sql.DB) usecase.DoctorRepository {
	return &DoctorRepo{
		db: db,
	}
}

func (r *DoctorRepo) Save(doctor *model.Doctor) error {
	query := `INSERT INTO doctors (id, full_name, specialization, email)
		VALUES ($1, $2, $3, $4);`
	_, err := r.db.Exec(query, doctor.ID, doctor.FullName, doctor.Specialization, doctor.Email)
	if err != nil {
		return err
	}
	return nil
}

func (r *DoctorRepo) GetAll() ([]*model.Doctor, error) {
	query := `SELECT * from doctors;`
	rows, err := r.db.Query(query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	defer rows.Close()

	doctors := make([]*model.Doctor, 0)
	for rows.Next() {
		doctor := &model.Doctor{}
		err = rows.Scan(
			&doctor.ID,
			&doctor.FullName,
			&doctor.Specialization,
			&doctor.Email,
			&doctor.CreateAt,
		)
		if err != nil {
			return nil, err
		}
		doctors = append(doctors, doctor)
	}

	return doctors, nil
}

func (r *DoctorRepo) GetByID(id string) (*model.Doctor, error) {
	query := `SELECT * FROM doctors
		where id = $1;`
	var doctor model.Doctor
	row := r.db.QueryRow(query, id)
	err := row.Scan(&doctor.ID, &doctor.FullName, &doctor.Specialization, &doctor.Email, &doctor.CreateAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("doctor not found")
		}
		return nil, err
	}
	return &doctor, nil
}

func (r *DoctorRepo) EmailExists(email string) (bool, error) {
	query := `SELECT count(*) FROM doctors
		where email = $1;`
	var cnt int
	row := r.db.QueryRow(query, email)
	err := row.Scan(&cnt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if cnt != 0 {
		return true, nil
	}
	return false, nil
}
