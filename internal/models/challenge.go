package model

import (
	"database/sql"
	"time"
)

type Challenge struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Category      string         `json:"category"` // DAILY, WEEKLY, etc.
	Type          string         `json:"type"`
	Variant       string         `json:"variant"`
	Difficulty    string         `json:"difficulty"`
	TargetReps    sql.NullInt32  `json:"targetReps,omitempty"`
	Duration      sql.NullInt32  `json:"duration,omitempty"`
	Sets          sql.NullInt32  `json:"sets,omitempty"`
	RepsPerSet    sql.NullInt32  `json:"repsPerSet,omitempty"`
	ImageURL      sql.NullString `json:"imageUrl,omitempty"`
	IconName      string         `json:"iconName"`
	IconColor     string         `json:"iconColor"`
	Participants  int            `json:"participants"`
	Completions   int            `json:"completions"`
	Likes         int            `json:"likes"`
	Points        int            `json:"points"`
	Badge         sql.NullString `json:"badge,omitempty"`
	StartDate     sql.NullTime   `json:"startDate,omitempty"`
	EndDate       sql.NullTime   `json:"endDate,omitempty"`
	Status        string         `json:"status"`
	UserCompleted sql.NullBool   `json:"userCompleted,omitempty"`
	UserLiked     sql.NullBool   `json:"userLiked,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
	IsOfficial    bool           `json:"isOfficial"`
	CreatedBy     sql.NullString `json:"createdBy,omitempty"`
	UpdatedBy     sql.NullString `json:"updatedBy,omitempty"`
	DeletedBy     sql.NullString `json:"deletedBy,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     sql.NullTime   `json:"deletedAt,omitempty"`
}

type UserChallengeProgress struct {
	ID          string       `json:"id"`
	ChallengeID string       `json:"challengeId"`
	UserID      string       `json:"userId"`
	Progress    int          `json:"progress"`
	CurrentReps int          `json:"currentReps"`
	TargetReps  int          `json:"targetReps"`
	Attempts    int          `json:"attempts"`
	CompletedAt sql.NullTime `json:"completedAt,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type ChallengeLike struct {
	ID          string    `json:"id"`
	ChallengeID string    `json:"challengeId"`
	UserID      string    `json:"userId"`
	CreatedAt   time.Time `json:"createdAt"`
}
