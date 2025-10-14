package scanner

import (
	"database/sql"
	"fmt"
	"time"

	model "github.com/MassBabyGeek/PumpPro-backend/internal/models"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/lib/pq"
)

// ScanUserProfile scanne une ligne SQL vers un UserProfile
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
	var updatedBy, createdBy, deletedBy sql.NullString
	var startDate, endDate, createdAt, updatedAt, deletedAt sql.NullTime
	var tagsNull sql.NullString
	var userCompleted, userLiked, userParticipated sql.NullBool

	err := scanner.Scan(
		&c.ID, &c.Title, &c.Description, &c.Category, &c.Type, &c.Variant, &c.Difficulty,
		&c.TargetReps, &c.Duration, &c.Sets, &c.RepsPerSet, &c.ImageURL,
		&c.IconName, &c.IconColor, &c.Participants, &c.Completions, &c.Likes, &c.Points,
		&c.Badge, &startDate, &endDate, &c.Status, &tagsNull, &c.IsOfficial,
		&createdBy, &updatedBy, &createdAt, &updatedAt, &deletedBy, &deletedAt,
		&userCompleted, &userLiked, &userParticipated,
	)
	if err != nil {
		return nil, err
	}

	c.Tags = utils.NullStringToStringArray(tagsNull)
	c.UpdatedBy = utils.NullStringToPointer(updatedBy)
	c.StartDate = utils.NullTimeToPointer(startDate)
	c.EndDate = utils.NullTimeToPointer(endDate)
	c.CreatedBy = utils.NullStringToPointer(createdBy)
	c.UpdatedBy = utils.NullStringToPointer(updatedBy)
	c.DeletedBy = utils.NullStringToPointer(deletedBy)
	c.UserCompleted = utils.NullBoolToBool(userCompleted)
	c.UserLiked = utils.NullBoolToBool(userLiked)
	c.UserParticipated = utils.NullBoolToBool(userParticipated)

	return &c, nil
}

func ScanChartData(scanner interface {
	Scan(dest ...interface{}) error
}) (model.ChartData, error) {

	var data model.ChartData
	var pushUps, duration sql.NullInt64
	var calories sql.NullFloat64
	var dateValue interface{}

	// Lecture des colonnes SQL
	if err := scanner.Scan(&dateValue, &pushUps, &duration, &calories); err != nil {
		return data, fmt.Errorf("could not scan chart data: %w", err)
	}

	// Conversion du champ "date"
	switch v := dateValue.(type) {
	case time.Time:
		data.Date = v.Format("2006-01-02")
	case string:
		data.Date = v
	default:
		data.Date = ""
	}

	// Conversion des valeurs nullables
	data.PushUps = utils.NullInt64ToInt(pushUps)
	data.Duration = utils.NullInt64ToInt(duration)
	data.Calories = utils.NullFloat64ToFloat64(calories)

	return data, nil
}

// ✅ ScanChallengeWithPqArray corrigée (NullTime-safe)
func ScanChallengeWithPqArray(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Challenge, error) {
	var c model.Challenge
	var startDate, endDate, createdAt, updatedAt, deletedAt sql.NullTime
	var createdBy, updatedBy, deletedBy sql.NullString

	err := scanner.Scan(
		&c.ID, &c.Title, &c.Description, &c.Category, &c.Type, &c.Variant, &c.Difficulty,
		&c.TargetReps, &c.Duration, &c.Sets, &c.RepsPerSet, &c.ImageURL,
		&c.IconName, &c.IconColor, &c.Participants, &c.Completions, &c.Likes, &c.Points,
		&c.Badge, &startDate, &endDate, &c.Status, pq.Array(&c.Tags), &c.IsOfficial,
		&createdBy, &updatedBy, &deletedBy, &createdAt, &updatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	c.CreatedBy = utils.NullStringToPointer(createdBy)
	c.UpdatedBy = utils.NullStringToPointer(updatedBy)
	c.DeletedBy = utils.NullStringToPointer(deletedBy)
	c.StartDate = utils.NullTimeToPointer(startDate)
	c.EndDate = utils.NullTimeToPointer(endDate)

	if createdAt.Valid {
		c.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		c.UpdatedAt = updatedAt.Time
	}
	if deletedAt.Valid {
		c.DeletedAt = deletedAt.Time
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

// ✅ ScanWorkoutProgramWithJSON corrigée (NullTime-safe)
func ScanWorkoutProgramWithJSON(scanner interface {
	Scan(dest ...interface{}) error
}, unmarshalJSON func([]byte, interface{}) error) (*model.WorkoutProgram, error) {
	var p model.WorkoutProgram
	var repsSequenceJSON []byte
	var createdAt, updatedAt, deletedAt sql.NullTime
	var createdBy, updatedBy, deletedBy sql.NullString

	err := scanner.Scan(
		&p.ID, &p.Name, &p.Description, &p.Type, &p.Variant, &p.Difficulty, &p.RestBetweenSets,
		&p.TargetReps, &p.TimeLimit, &p.Duration, &p.AllowRest, &p.Sets, &p.RepsPerSet,
		&repsSequenceJSON, &p.RepsPerMinute, &p.TotalMinutes,
		&p.IsCustom, &p.IsFeatured, &p.UsageCount,
		&createdBy, &updatedBy, &deletedBy,
		&createdAt, &updatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	p.CreatedBy = utils.NullStringToPointer(createdBy)
	p.UpdatedBy = utils.NullStringToPointer(updatedBy)
	p.DeletedBy = utils.NullStringToPointer(deletedBy)

	if createdAt.Valid {
		p.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		p.UpdatedAt = updatedAt.Time
	}
	if deletedAt.Valid {
		p.DeletedAt = deletedAt.Time
	}

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
	var createdAt, updatedAt, deletedAt sql.NullTime

	err := scanner.Scan(
		&p.ID, &p.Name, &p.Description, &p.Type, &p.Variant, &p.Difficulty,
		&p.RestBetweenSets, &p.TargetReps, &p.TimeLimit, &p.Duration, &p.AllowRest,
		&p.Sets, &p.RepsPerSet, &repsSequence, &p.RepsPerMinute, &p.TotalMinutes,
		&p.IsCustom, &p.IsFeatured, &p.UsageCount,
		&p.CreatedBy, &p.UpdatedBy, &p.DeletedBy,
		&createdAt, &updatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	if createdAt.Valid {
		p.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		p.UpdatedAt = updatedAt.Time
	}
	if deletedAt.Valid {
		p.DeletedAt = deletedAt.Time
	}

	if len(repsSequence) > 0 {
		p.RepsSequence = make([]int, len(repsSequence))
		for i, v := range repsSequence {
			p.RepsSequence[i] = int(v)
		}
	}

	return &p, nil
}

// ScanStats scanne une ligne SQL vers un Stats
func ScanStats(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Stats, error) {
	var stats model.Stats
	var totalWorkouts, totalPushUps, totalTime, bestSession sql.NullInt64
	var totalCalories, averagePushUps sql.NullFloat64

	err := scanner.Scan(
		&totalWorkouts, &totalPushUps, &totalTime, &bestSession, &totalCalories, &averagePushUps,
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
	var completedAt, createdAt, updatedAt sql.NullTime

	err := scanner.Scan(
		&progress.ID, &progress.ChallengeID, &progress.UserID, &progress.Progress,
		&progress.CurrentReps, &progress.TargetReps, &progress.Attempts,
		&completedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	progress.CompletedAt = utils.NullTimeToPointer(completedAt)
	if createdAt.Valid {
		progress.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		progress.UpdatedAt = updatedAt.Time
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
		&task.Score, &task.ScheduledDate, &task.IsLocked,
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
	}

	if createdAt.Valid {
		task.DateFields.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		task.DateFields.UpdatedAt = updatedAt.Time
	}
	if deletedAt.Valid {
		task.DateFields.DeletedAt = deletedAt.Time
	}

	return &task, nil
}

// ScanUserChallengeTaskProgress scanne une ligne SQL vers un UserChallengeTaskProgress
func ScanUserChallengeTaskProgress(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.UserChallengeTaskProgress, error) {
	var progress model.UserChallengeTaskProgress
	var completedAt, createdAt, updatedAt sql.NullTime

	err := scanner.Scan(
		&progress.ID, &progress.UserID, &progress.TaskID, &progress.ChallengeID,
		&progress.Completed, &completedAt, &progress.Score, &progress.Attempts,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	progress.CompletedAt = utils.NullTimeToPointer(completedAt)
	if createdAt.Valid {
		progress.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		progress.UpdatedAt = updatedAt.Time
	}

	return &progress, nil
}
