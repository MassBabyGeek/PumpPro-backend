package scanner

import (
	"database/sql"

	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/lib/pq"
)

// ScanUserProfile scanne une ligne SQL vers un UserProfile
// Utilise les types sql.Null* et les convertit automatiquement
func ScanUserProfile(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.UserProfile, error) {
	var user model.UserProfile
	var avatar, goal sql.NullString
	var age sql.NullInt64
	var weight, height sql.NullFloat64
	var updatedBy sql.NullString

	err := scanner.Scan(
		&user.ID, &user.Name, &user.Email, &avatar,
		&age, &weight, &height, &goal,
		&user.JoinDate, &user.CreatedAt, &user.UpdatedAt,
		&user.CreatedBy, &updatedBy,
	)
	if err != nil {
		return nil, err
	}

	// Conversions
	user.Avatar = utils.NullStringToString(avatar)
	user.Goal = utils.NullStringToString(goal)
	user.Age = utils.NullInt64ToInt(age)
	user.Weight = utils.NullFloat64ToFloat64(weight)
	user.Height = utils.NullFloat64ToFloat64(height)
	user.UpdatedBy = utils.NullStringToPointer(updatedBy)

	return &user, nil
}

// ScanChallenge scanne une ligne SQL vers un Challenge
func ScanChallenge(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Challenge, error) {
	var c model.Challenge
	var updatedBy sql.NullString
	var startDate, endDate sql.NullTime
	var tagsNull sql.NullString

	err := scanner.Scan(
		&c.ID, &c.Title, &c.Description, &c.Category, &c.Type, &c.Variant, &c.Difficulty,
		&c.TargetReps, &c.Duration, &c.Sets, &c.RepsPerSet, &c.ImageURL,
		&c.IconName, &c.IconColor, &c.Participants, &c.Completions, &c.Likes, &c.Points,
		&c.Badge, &startDate, &endDate, &c.Status, &tagsNull, &c.IsOfficial,
		&c.CreatedBy, &updatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Conversions
	c.Tags = utils.NullStringToStringArray(tagsNull)
	c.UpdatedBy = utils.NullStringToPointer(updatedBy)
	c.StartDate = utils.NullTimeToPointer(startDate)
	c.EndDate = utils.NullTimeToPointer(endDate)

	return &c, nil
}

// ScanChallengeWithPqArray scanne une ligne SQL vers un Challenge avec pq.Array pour les tags
func ScanChallengeWithPqArray(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Challenge, error) {
	var c model.Challenge

	err := scanner.Scan(
		&c.ID, &c.Title, &c.Description, &c.Category, &c.Type, &c.Variant, &c.Difficulty,
		&c.TargetReps, &c.Duration, &c.Sets, &c.RepsPerSet, &c.ImageURL,
		&c.IconName, &c.IconColor, &c.Participants, &c.Completions, &c.Likes, &c.Points,
		&c.Badge, &c.StartDate, &c.EndDate, &c.Status, pq.Array(&c.Tags), &c.IsOfficial,
		&c.CreatedBy, &c.UpdatedBy, &c.DeletedBy, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// ScanWorkoutSession scanne une ligne SQL vers un WorkoutSession
func ScanWorkoutSession(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.WorkoutSession, error) {
	var s model.WorkoutSession

	err := scanner.Scan(
		&s.ID, &s.ProgramID, &s.UserID, &s.StartTime, &s.EndTime,
		&s.TotalReps, &s.TotalDuration, &s.Completed, &s.Notes,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// ScanSetResult scanne une ligne SQL vers un SetResult
func ScanSetResult(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.SetResult, error) {
	var s model.SetResult

	err := scanner.Scan(
		&s.ID, &s.SessionID, &s.SetNumber, &s.TargetReps,
		&s.CompletedReps, &s.Duration, &s.Timestamp,
	)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// ScanWorkoutProgramWithJSON scanne une ligne SQL vers un WorkoutProgram
// Utilise []byte pour reps_sequence qui sera décodé en JSON
func ScanWorkoutProgramWithJSON(scanner interface {
	Scan(dest ...interface{}) error
}, unmarshalJSON func([]byte, interface{}) error) (*model.WorkoutProgram, error) {
	var p model.WorkoutProgram
	var repsSequenceJSON []byte

	err := scanner.Scan(
		&p.ID, &p.Name, &p.Description, &p.Type, &p.Variant, &p.Difficulty, &p.RestBetweenSets,
		&p.TargetReps, &p.TimeLimit, &p.Duration, &p.AllowRest, &p.Sets, &p.RepsPerSet,
		&repsSequenceJSON, &p.RepsPerMinute, &p.TotalMinutes,
		&p.IsCustom, &p.IsFeatured, &p.UsageCount,
		&p.CreatedBy, &p.UpdatedBy, &p.DeletedBy, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
	)
	if err != nil {
		return nil, err
	}

	// Décoder reps_sequence si présent
	if repsSequenceJSON != nil {
		unmarshalJSON(repsSequenceJSON, &p.RepsSequence)
	}

	return &p, nil
}

// ScanWorkoutProgram scanne une ligne SQL vers un WorkoutProgram avec pq.Array
func ScanWorkoutProgram(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.WorkoutProgram, error) {
	var p model.WorkoutProgram
	var repsSequence pq.Int64Array

	err := scanner.Scan(
		&p.ID, &p.Name, &p.Description, &p.Type, &p.Variant, &p.Difficulty,
		&p.RestBetweenSets, &p.TargetReps, &p.TimeLimit, &p.Duration, &p.AllowRest,
		&p.Sets, &p.RepsPerSet, &repsSequence, &p.RepsPerMinute, &p.TotalMinutes,
		&p.IsCustom, &p.IsFeatured, &p.UsageCount,
		&p.CreatedBy, &p.UpdatedBy, &p.DeletedBy,
		&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
	)
	if err != nil {
		return nil, err
	}

	// Conversion de pq.Int64Array vers []int
	if len(repsSequence) > 0 {
		p.RepsSequence = make([]int, len(repsSequence))
		for i, v := range repsSequence {
			p.RepsSequence[i] = int(v)
		}
	}

	return &p, nil
}

// ScanStats scanne une ligne SQL vers un Stats
// Utilise directement les types sql.Null* et les convertit
func ScanStats(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Stats, error) {
	var stats model.Stats
	var totalWorkouts, totalPushUps, totalTime, bestSession sql.NullInt64
	var totalCalories, averagePushUps sql.NullFloat64

	err := scanner.Scan(
		&totalWorkouts,
		&totalPushUps,
		&totalTime,
		&bestSession,
		&totalCalories,
		&averagePushUps,
	)
	if err != nil {
		return nil, err
	}

	stats.TotalWorkouts = utils.NullInt64ToInt(totalWorkouts)
	stats.TotalPushUps = utils.NullInt64ToInt(totalPushUps)
	stats.TotalTime = utils.NullInt64ToInt(totalTime)
	stats.BestSession = utils.NullInt64ToInt(bestSession)
	stats.TotalCalories = utils.NullFloat64ToFloat64(totalCalories)
	stats.AveragePushUps = utils.NullFloat64ToFloat64(averagePushUps)

	return &stats, nil
}

// ScanUserChallengeProgress scanne une ligne SQL vers un UserChallengeProgress
func ScanUserChallengeProgress(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.UserChallengeProgress, error) {
	var progress model.UserChallengeProgress

	err := scanner.Scan(
		&progress.ID, &progress.ChallengeID, &progress.UserID, &progress.Progress,
		&progress.CurrentReps, &progress.TargetReps, &progress.Attempts, &progress.CompletedAt,
		&progress.CreatedAt, &progress.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &progress, nil
}

// ScanChallengeTask scanne une ligne SQL vers un ChallengeTask
func ScanChallengeTask(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.ChallengeTask, error) {
	var task model.ChallengeTask
	var createdBy, deletedBy, updatedBy sql.NullString
	var createdAt, deletedAt, updatedAt sql.NullTime

	err := scanner.Scan(
		&task.ID, &task.ChallengeID, &task.Day, &task.Title, &task.Description,
		&task.Type, &task.Variant, &task.TargetReps, &task.Duration, &task.Sets, &task.RepsPerSet,
		&task.ScheduledDate, &task.IsLocked,
		&createdBy, &updatedBy, &deletedBy,
		&createdAt, &updatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	task.DateFields = model.DateFields{
		CreatedBy: utils.NullStringToPointer(createdBy),
		UpdatedBy: utils.NullStringToPointer(updatedBy),
		DeletedBy: utils.NullStringToPointer(deletedBy),
		CreatedAt: *utils.NullTimeToPointer(createdAt),
		UpdatedAt: *utils.NullTimeToPointer(updatedAt),
		DeletedAt: *utils.NullTimeToPointer(deletedAt),
	}

	return &task, nil
}

// ScanUserChallengeTaskProgress scanne une ligne SQL vers un UserChallengeTaskProgress
func ScanUserChallengeTaskProgress(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.UserChallengeTaskProgress, error) {
	var progress model.UserChallengeTaskProgress

	err := scanner.Scan(
		&progress.ID, &progress.UserID, &progress.TaskID, &progress.ChallengeID,
		&progress.Completed, &progress.CompletedAt, &progress.Score, &progress.Attempts,
		&progress.CreatedAt, &progress.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &progress, nil
}
