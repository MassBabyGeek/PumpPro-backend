package model

import "time"

// AdminDashboardStats contient toutes les statistiques pour le dashboard admin
type AdminDashboardStats struct {
	// Statistiques générales
	TotalUsers          int     `json:"totalUsers"`
	ActiveUsers         int     `json:"activeUsers"`          // Utilisateurs actifs dans les dernières 24h
	TotalChallenges     int     `json:"totalChallenges"`
	ActiveChallenges    int     `json:"activeChallenges"`
	TotalPrograms       int     `json:"totalPrograms"`
	TotalWorkouts       int     `json:"totalWorkouts"`
	TotalPushups        int     `json:"totalPushups"`
	TotalPhotos         int     `json:"totalPhotos"`
	TotalBugReports     int     `json:"totalBugReports"`
	PendingBugReports   int     `json:"pendingBugReports"`

	// Statistiques de croissance
	NewUsersToday       int     `json:"newUsersToday"`
	NewUsersThisWeek    int     `json:"newUsersThisWeek"`
	NewUsersThisMonth   int     `json:"newUsersThisMonth"`

	// Engagement
	WorkoutsToday       int     `json:"workoutsToday"`
	WorkoutsThisWeek    int     `json:"workoutsThisWeek"`
	WorkoutsThisMonth   int     `json:"workoutsThisMonth"`

	// Moyennes
	AvgPushupsPerUser   float64 `json:"avgPushupsPerUser"`
	AvgWorkoutsPerUser  float64 `json:"avgWorkoutsPerUser"`

	// Stockage
	StorageUsed         string  `json:"storageUsed"` // En MB ou GB

	// Timestamp
	GeneratedAt         time.Time `json:"generatedAt"`
}

// AdminUserActivity représente l'activité récente des utilisateurs
type AdminUserActivity struct {
	UserID      string    `json:"userId"`
	UserName    string    `json:"userName"`
	UserAvatar  *string   `json:"userAvatar,omitempty"`
	Action      string    `json:"action"` // "workout", "challenge_started", "challenge_completed", "signup"
	EntityType  *string   `json:"entityType,omitempty"` // "program", "challenge", etc.
	EntityID    *string   `json:"entityId,omitempty"`
	EntityName  *string   `json:"entityName,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// AdminSystemHealth représente la santé du système
type AdminSystemHealth struct {
	Status           string            `json:"status"` // "healthy", "warning", "critical"
	DatabaseStatus   string            `json:"databaseStatus"`
	DatabaseSize     string            `json:"databaseSize"`
	ResponseTime     float64           `json:"responseTime"` // En ms
	ErrorRate        float64           `json:"errorRate"`    // Pourcentage
	ActiveSessions   int               `json:"activeSessions"`
	Uptime           string            `json:"uptime"`
	Memory           map[string]string `json:"memory"` // used, total, percentage
	CheckedAt        time.Time         `json:"checkedAt"`
}

// AdminTopContent contient les contenus les plus populaires
type AdminTopContent struct {
	Challenges []TopItem `json:"challenges"`
	Programs   []TopItem `json:"programs"`
	Users      []TopItem `json:"users"`
}

type TopItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Avatar      *string `json:"avatar,omitempty"`
	ImageURL    *string `json:"imageUrl,omitempty"`
	Count       int     `json:"count"` // likes, completions, ou workouts selon le contexte
	Metric      string  `json:"metric"` // "likes", "completions", "workouts"
	MetricValue int     `json:"metricValue"`
}

// AdminAnalytics contient les données analytiques pour les graphiques
type AdminAnalytics struct {
	UserGrowth      []DataPoint `json:"userGrowth"`
	WorkoutActivity []DataPoint `json:"workoutActivity"`
	ChallengeStats  []DataPoint `json:"challengeStats"`
}

type DataPoint struct {
	Date  string  `json:"date"`
	Value int     `json:"value"`
	Label *string `json:"label,omitempty"`
}

// AdminUserManagement pour la gestion des utilisateurs
type AdminUserListItem struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	Avatar          *string    `json:"avatar,omitempty"`
	IsAdmin         bool       `json:"isAdmin"`
	Score           int        `json:"score"`
	TotalWorkouts   int        `json:"totalWorkouts"`
	TotalPushups    int        `json:"totalPushups"`
	JoinDate        time.Time  `json:"joinDate"`
	LastActive      *time.Time `json:"lastActive,omitempty"`
	Status          string     `json:"status"` // "active", "banned", "deleted"
}
