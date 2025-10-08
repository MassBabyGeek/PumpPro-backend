package model

import (
	"database/sql"
	"time"
)

type WorkoutProgram struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     sql.NullString `json:"description,omitempty"`
	Type            string         `json:"type"` // FREE_MODE, TARGET_REPS, MAX_TIME, SETS_REPS, PYRAMID, EMOM, AMRAP
	Variant         string         `json:"variant"` // STANDARD, INCLINE, DECLINE, DIAMOND, WIDE, PIKE, ARCHER
	Difficulty      string         `json:"difficulty"` // BEGINNER, INTERMEDIATE, ADVANCED
	RestBetweenSets sql.NullInt32  `json:"restBetweenSets,omitempty"`

	// Champs spécifiques selon le type
	TargetReps      sql.NullInt32  `json:"targetReps,omitempty"`      // Pour TARGET_REPS
	TimeLimit       sql.NullInt32  `json:"timeLimit,omitempty"`       // Pour TARGET_REPS (optionnel)
	Duration        sql.NullInt32  `json:"duration,omitempty"`        // Pour MAX_TIME, AMRAP
	AllowRest       sql.NullBool   `json:"allowRest,omitempty"`       // Pour MAX_TIME
	Sets            sql.NullInt32  `json:"sets,omitempty"`            // Pour SETS_REPS
	RepsPerSet      sql.NullInt32  `json:"repsPerSet,omitempty"`      // Pour SETS_REPS
	RepsSequence    []int          `json:"repsSequence,omitempty"`    // Pour PYRAMID
	RepsPerMinute   sql.NullInt32  `json:"repsPerMinute,omitempty"`   // Pour EMOM
	TotalMinutes    sql.NullInt32  `json:"totalMinutes,omitempty"`    // Pour EMOM

	IsCustom        bool           `json:"isCustom"`
	IsFeatured      bool           `json:"isFeatured"`
	UsageCount      int            `json:"usageCount"` // Nombre de fois utilisé

	CreatedBy       sql.NullString `json:"createdBy,omitempty"`
	UpdatedBy       sql.NullString `json:"updatedBy,omitempty"`
	DeletedBy       sql.NullString `json:"deletedBy,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       sql.NullTime   `json:"deletedAt,omitempty"`
}

type WorkoutSession struct {
	ID           string         `json:"sessionId"`
	ProgramID    string         `json:"programId"`
	UserID       string         `json:"userId"`
	StartTime    time.Time      `json:"startTime"`
	EndTime      sql.NullTime   `json:"endTime,omitempty"`
	TotalReps    int            `json:"totalReps"`
	TotalDuration int           `json:"totalDuration"` // en secondes
	Completed    bool           `json:"completed"`
	Notes        sql.NullString `json:"notes,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	Sets         []SetResult    `json:"sets"`
}

type SetResult struct {
	ID            string       `json:"id"`
	SessionID     string       `json:"sessionId"`
	SetNumber     int          `json:"setNumber"`
	TargetReps    sql.NullInt32 `json:"targetReps,omitempty"`
	CompletedReps int          `json:"completedReps"`
	Duration      int          `json:"duration"` // en secondes
	Timestamp     time.Time    `json:"timestamp"`
}
