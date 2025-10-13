package model

import (
	"database/sql"
	"time"
)

type Challenge struct {
	ID            string           `json:"id"`
	Title         string           `json:"title"`
	Description   string           `json:"description"`
	Category      string           `json:"category"` // DAILY, WEEKLY, etc.
	Type          string           `json:"type"`
	Variant       string           `json:"variant"`
	Difficulty    string           `json:"difficulty"`
	TargetReps    *int             `json:"targetReps,omitempty"`
	Duration      *int             `json:"duration,omitempty"`
	Sets          *int             `json:"sets,omitempty"`
	RepsPerSet    *int             `json:"repsPerSet,omitempty"`
	ImageURL      *string          `json:"imageUrl,omitempty"`
	IconName      string           `json:"iconName"`
	IconColor     string           `json:"iconColor"`
	Participants  int              `json:"participants"`
	Completions   int              `json:"completions"`
	Likes         int              `json:"likes"`
	Points        int              `json:"points"`
	Badge         *string          `json:"badge,omitempty"`
	StartDate     *time.Time       `json:"startDate,omitempty"`
	EndDate       *time.Time       `json:"endDate,omitempty"`
	Status        string           `json:"status"`
	UserCompleted sql.NullBool     `json:"userCompleted,omitempty"`
	UserLiked     sql.NullBool     `json:"userLiked,omitempty"`
	Tags          []string         `json:"tags,omitempty"`
	IsOfficial    bool             `json:"isOfficial"`
	Tasks         []ChallengeTask  `json:"challengeTasks,omitempty"`
	CreatedBy     *string          `json:"createdBy,omitempty"`
	UpdatedBy     *string          `json:"updatedBy,omitempty"`
	DeletedBy     *string          `json:"deletedBy,omitempty"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
	DeletedAt     *time.Time       `json:"deletedAt,omitempty"`
}

type ChallengeTask struct {
	ID            string     `json:"id"`
	ChallengeID   string     `json:"challengeId"`
	Day           int        `json:"day"`
	Title         string     `json:"title"`
	Description   *string    `json:"description,omitempty"`
	Type          *string    `json:"type,omitempty"`
	Variant       *string    `json:"variant,omitempty"`
	TargetReps    *int       `json:"targetReps,omitempty"`
	Duration      *int       `json:"duration,omitempty"`
	Sets          *int       `json:"sets,omitempty"`
	RepsPerSet    *int       `json:"repsPerSet,omitempty"`
	Completed     bool       `json:"completed"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	Score         *int       `json:"score,omitempty"`
	ScheduledDate *time.Time `json:"scheduledDate,omitempty"`
	IsLocked      bool       `json:"isLocked"`
	CreatedBy     *string    `json:"createdBy,omitempty"`
	UpdatedBy     *string    `json:"updatedBy,omitempty"`
	DeletedBy     *string    `json:"deletedBy,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
}

type UserChallengeProgress struct {
	ID          string     `json:"id"`
	ChallengeID string     `json:"challengeId"`
	UserID      string     `json:"userId"`
	Progress    int        `json:"progress"`
	CurrentReps int        `json:"currentReps"`
	TargetReps  int        `json:"targetReps"`
	Attempts    int        `json:"attempts"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type ChallengeLike struct {
	ID          string    `json:"id"`
	ChallengeID string    `json:"challengeId"`
	UserID      string    `json:"userId"`
	CreatedAt   time.Time `json:"createdAt"`
}
