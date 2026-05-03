package repository

import (
	"database/sql"
	"errors"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"
)

type appointmentRepo struct {
	db *sql.DB
}

func New(db *sql.DB) usecase.AppointmentRepository {
	return &appointmentRepo{
		db: db,
	}
}

func (r *appointmentRepo) Save(a *model.Appointment) error {
	query := `
		INSERT INTO appointments (id, title, description, doctor_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.Exec(query, a.ID, a.Title, a.Description, a.DoctorID, a.Status, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *appointmentRepo) GetByID(id string) (*model.Appointment, error) {
	query := `
		SELECT id, title, description, doctor_id, status, created_at, updated_at
		FROM appointments
		WHERE id = $1`

	row := r.db.QueryRow(query, id)

	var a model.Appointment
	err := row.Scan(&a.ID, &a.Title, &a.Description, &a.DoctorID, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("appointment not found")
		}
		return nil, err
	}
	return &a, nil
}

func (r *appointmentRepo) GetAll() ([]*model.Appointment, error) {
	query := `SELECT id, title, description, doctor_id, status, created_at, updated_at FROM appointments`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.Appointment
	for rows.Next() {
		a := &model.Appointment{}
		err := rows.Scan(
			&a.ID,
			&a.Title,
			&a.Description,
			&a.DoctorID,
			&a.Status,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *appointmentRepo) Update(a *model.Appointment) error {
	query := `
		UPDATE appointments
		SET title = $1, description = $2, doctor_id = $3, status = $4, updated_at = now()
		WHERE id = $5`

	res, err := r.db.Exec(query, a.Title, a.Description, a.DoctorID, a.Status, a.ID)
	if err != nil {
		return err
	}

	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("appointment not found")
	}

	return nil
}
