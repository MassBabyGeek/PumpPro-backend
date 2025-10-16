package api

import (
	"net/http"

	"github.com/MassBabyGeek/PumpPro-backend/internal/handler"
	"github.com/MassBabyGeek/PumpPro-backend/internal/middleware"
	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/fatih/color"
	"github.com/gorilla/mux"
)

func SetupRouter() http.Handler {
	r := mux.NewRouter()
	r.Use(middleware.OptionalAuth)

	authenticatedRoutes := r.PathPrefix("/").Subrouter()
	authenticatedRoutes.Use(middleware.AuthMiddleware)
	authenticatedRoutes.Use(middleware.LoggerMiddleware)

	// Root - API documentation
	r.HandleFunc("/", handler.RootHandler).Methods(http.MethodGet)

	// Auth
	r.HandleFunc("/auth/login", handler.Login).Methods(http.MethodPost)
	authenticatedRoutes.HandleFunc("/auth/logout", handler.Logout).Methods(http.MethodPost)
	r.HandleFunc("/auth/signup", handler.Signup).Methods(http.MethodPost)
	r.HandleFunc("/auth/register", handler.Register).Methods(http.MethodPost)
	r.HandleFunc("/auth/reset-password", handler.ResetPassword).Methods(http.MethodPost)
	r.HandleFunc("/auth/verify-email", handler.VerifyEmail).Methods(http.MethodPost)
	r.HandleFunc("/auth/google", handler.GoogleAuth).Methods(http.MethodPost)
	r.HandleFunc("/auth/apple", handler.AppleAuth).Methods(http.MethodPost)

	// Users
	r.HandleFunc("/users", handler.CreateUser).Methods(http.MethodPost)
	r.HandleFunc("/users", handler.GetUsers).Methods(http.MethodGet)
	r.HandleFunc("/users/{id}", handler.GetUser).Methods(http.MethodGet)
	authenticatedRoutes.HandleFunc("/users/{id}", handler.DeleteUser).Methods(http.MethodDelete)
	authenticatedRoutes.HandleFunc("/users/{id}/avatar", handler.UploadAvatar).Methods(http.MethodPost)
	authenticatedRoutes.HandleFunc("/users/{userId}/stats/{period}", handler.GetUserStats).Methods(http.MethodGet)
	authenticatedRoutes.HandleFunc("/users/{userId}/charts/{period}", handler.GetChartData).Methods(http.MethodGet)
	authenticatedRoutes.HandleFunc("/users/{id}", handler.UpdateUser).Methods(http.MethodPut, http.MethodPatch)

	// Challenges
	r.HandleFunc("/challenges", handler.GetChallenges).Methods(http.MethodGet)
	r.HandleFunc("/challenges/{id}", handler.GetChallengeById).Methods(http.MethodGet)
	authenticatedRoutes.HandleFunc("/challenges", handler.CreateChallenge).Methods(http.MethodPost)
	authenticatedRoutes.HandleFunc("/challenges/{id}", handler.UpdateChallenge).Methods(http.MethodPut)
	authenticatedRoutes.HandleFunc("/challenges/{id}", handler.DeleteChallenge).Methods(http.MethodDelete)
	authenticatedRoutes.HandleFunc("/challenges/{id}/tasks/{taskId}", handler.CompleteTask).Methods(http.MethodPost)

	// Challenge interactions
	authenticatedRoutes.HandleFunc("/challenges/{id}/like", handler.LikeChallenge).Methods(http.MethodPost)
	authenticatedRoutes.HandleFunc("/challenges/{id}/like", handler.UnlikeChallenge).Methods(http.MethodDelete)
	authenticatedRoutes.HandleFunc("/challenges/{id}/start", handler.StartChallenge).Methods(http.MethodPost)
	authenticatedRoutes.HandleFunc("/challenges/{id}/complete", handler.CompleteChallenge).Methods(http.MethodPost)
	authenticatedRoutes.HandleFunc("/challenges/{id}/progress", handler.GetUserChallengeProgress).Methods(http.MethodGet)

	// User challenges
	r.HandleFunc("/users/{userId}/challenges/active", handler.GetUserActiveChallenges).Methods(http.MethodGet)
	r.HandleFunc("/users/{userId}/challenges/completed", handler.GetUserCompletedChallenges).Methods(http.MethodGet)

	// Programs
	r.HandleFunc("/programs", handler.GetPrograms).Methods(http.MethodGet)
	r.HandleFunc("/programs/{id}", handler.GetProgramById).Methods(http.MethodGet)
	r.HandleFunc("/programs", handler.CreateProgram).Methods(http.MethodPost)
	r.HandleFunc("/programs/{id}", handler.UpdateProgram).Methods(http.MethodPatch, http.MethodPut)
	r.HandleFunc("/programs/{id}", handler.DeleteProgram).Methods(http.MethodDelete)

	// Program specific routes
	r.HandleFunc("/programs/featured", handler.GetFeaturedPrograms).Methods(http.MethodGet)
	r.HandleFunc("/programs/popular", handler.GetPopularPrograms).Methods(http.MethodGet)
	r.HandleFunc("/programs/{id}/duplicate", handler.DuplicateProgram).Methods(http.MethodPost)

	// User programs
	r.HandleFunc("/users/{userId}/programs", handler.GetUserCustomPrograms).Methods(http.MethodGet)
	r.HandleFunc("/users/{userId}/programs/recommended", handler.GetRecommendedPrograms).Methods(http.MethodGet)

	// Programs by difficulty
	r.HandleFunc("/programs/difficulty/{difficulty}", handler.GetProgramsByDifficulty).Methods(http.MethodGet)

	// Workout Sessions
	r.HandleFunc("/workouts", handler.GetWorkoutSessions).Methods(http.MethodGet)
	authenticatedRoutes.HandleFunc("/workouts", handler.SaveWorkoutSession).Methods(http.MethodPost)
	r.HandleFunc("/workouts/{id}", handler.GetWorkoutSession).Methods(http.MethodGet)
	authenticatedRoutes.HandleFunc("/workouts/{id}", handler.UpdateWorkoutSession).Methods(http.MethodPatch)
	authenticatedRoutes.HandleFunc("/workouts/{id}", handler.DeleteWorkoutSession).Methods(http.MethodDelete)

	// User workout sessions
	r.HandleFunc("/users/{userId}/workouts", handler.GetUsersWorkoutSessions).Methods(http.MethodGet)
	r.HandleFunc("/users/{userId}/workouts/stats", handler.GetWorkoutStats).Methods(http.MethodGet)
	r.HandleFunc("/users/{userId}/workouts/summary", handler.GetWorkoutSummary).Methods(http.MethodGet)
	r.HandleFunc("/users/{userId}/workouts/records", handler.GetPersonalRecords).Methods(http.MethodGet)

	// Set results
	r.HandleFunc("/workouts/{sessionId}/sets", handler.SaveSetResults).Methods(http.MethodPost)
	r.HandleFunc("/workouts/{sessionId}/sets", handler.GetSetResults).Methods(http.MethodGet)

	// Leaderboard
	r.HandleFunc("/leaderboard", handler.GetLeaderboard).Methods(http.MethodGet)
	r.HandleFunc("/leaderboard/top", handler.GetTopPerformers).Methods(http.MethodGet)
	r.HandleFunc("/leaderboard/users/{userId}", handler.GetUserRank).Methods(http.MethodGet)
	r.HandleFunc("/leaderboard/users/{userId}/nearby", handler.GetNearbyUsers).Methods(http.MethodGet)

	// Challenge leaderboard
	r.HandleFunc("/challenges/{challengeId}/leaderboard", handler.GetChallengeLeaderboard).Methods(http.MethodGet)

	// Friends leaderboard
	r.HandleFunc("/users/{userId}/friends/leaderboard", handler.GetFriendsLeaderboard).Methods(http.MethodGet)

	// Health check
	r.HandleFunc("/health", handler.HealthCheck).Methods(http.MethodGet)

	// Bug reports / Signalements
	r.HandleFunc("/bug-reports", handler.CreateBugReport).Methods(http.MethodPost)
	r.HandleFunc("/bug-reports", handler.GetBugReports).Methods(http.MethodGet)
	r.HandleFunc("/bug-reports/stats", handler.GetBugReportStats).Methods(http.MethodGet)
	r.HandleFunc("/bug-reports/{id}", handler.GetBugReportById).Methods(http.MethodGet)
	authenticatedRoutes.HandleFunc("/bug-reports/{id}", handler.UpdateBugReport).Methods(http.MethodPut, http.MethodPatch)
	authenticatedRoutes.HandleFunc("/bug-reports/{id}", handler.DeleteBugReport).Methods(http.MethodDelete)

	// Likes system (générique)
	authenticatedRoutes.HandleFunc("/likes/{entityType}/{entityId}/toggle", handler.ToggleLike).Methods(http.MethodPost)
	r.HandleFunc("/likes/{entityType}/{entityId}", handler.GetLikeStatus).Methods(http.MethodGet)
	r.HandleFunc("/likes/users/{userId}", handler.GetUserLikedEntities).Methods(http.MethodGet)
	r.HandleFunc("/likes/top", handler.GetTopLiked).Methods(http.MethodGet)

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		utils.LogError("404 Not Found: %s %s", r.Method, r.URL.Path)
		color.Yellow("[404] %s %s (route non trouvée)", r.Method, r.URL.Path)
		http.Error(w, "Route not found", http.StatusNotFound)
	})

	return r
}
