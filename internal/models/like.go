package model

import "time"

// EntityType représente les types d'entités qui peuvent être likées
type EntityType string

const (
	EntityTypeChallenge EntityType = "challenge"
	EntityTypeProgram   EntityType = "program"
	EntityTypeWorkout   EntityType = "workout"
	EntityTypeComment   EntityType = "comment"
)

// Like représente un like d'un utilisateur sur une entité
type Like struct {
	ID         string     `json:"id"`
	UserID     string     `json:"userId"`
	EntityType EntityType `json:"entityType"`
	EntityID   string     `json:"entityId"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// LikeInfo contient les informations de like pour une entité donnée
type LikeInfo struct {
	TotalLikes int  `json:"totalLikes"`
	UserLiked  bool `json:"userLiked"`
}

// LikesCount représente le nombre de likes pour une entité
type LikesCount struct {
	EntityType EntityType `json:"entityType"`
	EntityID   string     `json:"entityId"`
	TotalLikes int        `json:"totalLikes"`
}
