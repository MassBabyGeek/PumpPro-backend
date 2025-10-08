package model

type LeaderboardEntry struct {
	UserID   string   `json:"userId"`
	UserName string   `json:"userName"`
	Avatar   *string  `json:"avatar,omitempty"`
	Rank     int      `json:"rank"`
	Score    int      `json:"score"`
	Change   *int     `json:"change,omitempty"`
	Badges   []string `json:"badges,omitempty"`
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
