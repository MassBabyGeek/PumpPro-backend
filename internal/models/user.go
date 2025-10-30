package model

import (
	"time"
)

// AuditFields contient les champs d'audit standard pour toutes les entités
type DateFields struct {
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
	Score    int       `json:"score"`
	IsAdmin  bool      `json:"isAdmin"`
	JoinDate time.Time `json:"joinDate,omitempty"`
	DateFields
}

type ChartData struct {
	Date     string  `json:"date"`
	PushUps  int     `json:"pushUps"`
	Duration int     `json:"duration"`
	Calories float64 `json:"calories"`
}

// UserCreator contient les informations de l'utilisateur créateur d'une entité
type UserCreator struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}
