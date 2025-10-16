package model

import (
	"database/sql"
	"time"
)

type WorkoutProgram struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	Type            string  `json:"type"`       // FREE_MODE, TARGET_REPS, MAX_TIME, SETS_REPS, PYRAMID, EMOM, AMRAP
	Variant         string  `json:"variant"`    // STANDARD, INCLINE, DECLINE, DIAMOND, WIDE, PIKE, ARCHER
	Difficulty      string  `json:"difficulty"` // BEGINNER, INTERMEDIATE, ADVANCED
	RestBetweenSets *int    `json:"restBetweenSets,omitempty"`

	// Champs spécifiques selon le type
	TargetReps    *int         `json:"targetReps,omitempty"`    // Pour TARGET_REPS
	TimeLimit     *int         `json:"timeLimit,omitempty"`     // Pour TARGET_REPS (optionnel)
	Duration      *int         `json:"duration,omitempty"`      // Pour MAX_TIME, AMRAP
	AllowRest     sql.NullBool `json:"allowRest,omitempty"`     // Pour MAX_TIME
	Sets          *int         `json:"sets,omitempty"`          // Pour SETS_REPS
	RepsPerSet    *int         `json:"repsPerSet,omitempty"`    // Pour SETS_REPS
	RepsSequence  []int        `json:"repsSequence,omitempty"`  // Pour PYRAMID
	RepsPerMinute *int         `json:"repsPerMinute,omitempty"` // Pour EMOM
	TotalMinutes  *int         `json:"totalMinutes,omitempty"`  // Pour EMOM

	IsCustom   bool `json:"isCustom"`
	IsFeatured bool `json:"isFeatured"`
	UsageCount int  `json:"usageCount"` // Nombre de fois utilisé
	Likes      int  `json:"likes"`
	UserLiked  bool `json:"userLiked,omitempty"`

	DateFields
}

type WorkoutSession struct {
	ID              string      `json:"sessionId"`
	ProgramID       string      `json:"programId"`
	UserID          string      `json:"userId"`
	ChallengeID     *string     `json:"challengeId,omitempty"`
	ChallengeTaskID *string     `json:"challengeTaskId,omitempty"`
	StartTime       time.Time   `json:"startTime"`
	EndTime         *time.Time  `json:"endTime,omitempty"`
	TotalReps       int         `json:"totalReps"`
	TotalDuration   int         `json:"totalDuration"` // en secondes
	Completed       bool        `json:"completed"`
	Notes           *string     `json:"notes,omitempty"`
	Sets            []SetResult `json:"sets"`
	Likes           int         `json:"likes"`
	UserLiked       bool        `json:"userLiked,omitempty"`

	DateFields
}

type SetResult struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"sessionId"`
	SetNumber     int       `json:"setNumber"`
	TargetReps    *int      `json:"targetReps,omitempty"`
	CompletedReps int       `json:"completedReps"`
	Duration      int       `json:"duration"` // en secondes
	Timestamp     time.Time `json:"timestamp"`
}

type Stats struct {
	TotalWorkouts  int     `json:"totalWorkouts"`
	TotalPushUps   int     `json:"totalPushUps"`
	TotalCalories  float64 `json:"totalCalories"`
	TotalTime      int     `json:"totalTime"`
	BestSession    int     `json:"bestSession"`
	AveragePushUps float64 `json:"averagePushUps"`
}
