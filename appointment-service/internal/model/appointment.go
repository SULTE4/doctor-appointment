package model

import "time"

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

func (s Status) IsValid() bool {
	return s == StatusNew || s == StatusInProgress || s == StatusDone
}

type Appointment struct {
	ID          string
	Title       string
	Description string
	DoctorID    string
	Status      Status // define a custom Status type
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
