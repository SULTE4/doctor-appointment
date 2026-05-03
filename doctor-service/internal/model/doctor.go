package model

import "time"

type Doctor struct {
	ID             string    `json:"id"`
	FullName       string    `json:"full_name"`
	Specialization string    `json:"specialization"`
	Email          string    `json:"email"`
	CreateAt       time.Time `json:"createdAt"`
}
