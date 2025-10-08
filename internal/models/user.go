package model

import (
	"time"
)

// AuditFields contient les champs d'audit standard pour toutes les entités
type AuditFields struct {
	CreatedBy *string   `json:"createdBy,omitempty"`
	UpdatedBy *string   `json:"updatedBy,omitempty"`
	DeletedAt time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string   `json:"deletedBy,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

type UserProfile struct {
	ID       string    `json:"id,omitempty"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Avatar   string    `json:"avatar,omitempty"`
	Age      int       `json:"age,omitempty"`
	Weight   float64   `json:"weight,omitempty"`
	Height   float64   `json:"height,omitempty"`
	Goal     string    `json:"goal,omitempty"`
	Provider string    `json:"provider,omitempty"` // email, google, apple
	JoinDate time.Time `json:"joinDate,omitempty"`
	AuditFields
}
