package model

import (
	"database/sql"
)

type LeaderboardEntry struct {
	UserID   string         `json:"userId"`
	UserName string         `json:"userName"`
	Avatar   sql.NullString `json:"avatar,omitempty"`
	Rank     int            `json:"rank"`
	Score    int            `json:"score"` // Total push-ups, points, etc.
	Change   sql.NullInt32  `json:"change,omitempty"` // Changement de position
	Badges   []string       `json:"badges,omitempty"`
}

type UserRank struct {
	UserID     string  `json:"userId"`
	Rank       int     `json:"rank"`
	Score      int     `json:"score"`
	TotalUsers int     `json:"totalUsers"`
	Percentile float64 `json:"percentile"` // Top X%
}

type LeaderboardCache struct {
	ID        string `json:"id"`
	Period    string `json:"period"` // daily, weekly, monthly, all-time
	UserID    string `json:"userId"`
	Score     int    `json:"score"`
	Rank      int    `json:"rank"`
	Change    int    `json:"change"`
	UpdatedAt string `json:"updatedAt"`
}
