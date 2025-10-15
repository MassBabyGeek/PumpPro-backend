package handler

import (
	"net/http"

	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
)

// RootHandler affiche toutes les routes disponibles de l'API
func RootHandler(w http.ResponseWriter, r *http.Request) {
	routes := map[string]interface{}{
		"name":    "PumpPro API",
		"version": "1.0.0",
		"status":  "running",
		"routes": map[string]interface{}{
			"auth": []map[string]string{
				{"method": "POST", "path": "/auth/login", "description": "Connexion utilisateur"},
				{"method": "POST", "path": "/auth/logout", "description": "Déconnexion utilisateur"},
				{"method": "POST", "path": "/auth/signup", "description": "Inscription utilisateur"},
				{"method": "POST", "path": "/auth/register", "description": "Inscription utilisateur (alias)"},
				{"method": "POST", "path": "/auth/reset-password", "description": "Réinitialiser le mot de passe"},
				{"method": "POST", "path": "/auth/verify-email", "description": "Vérifier l'email"},
				{"method": "POST", "path": "/auth/google", "description": "Authentification Google OAuth"},
				{"method": "POST", "path": "/auth/apple", "description": "Authentification Apple Sign In"},
			},
			"users": []map[string]string{
				{"method": "GET", "path": "/users", "description": "Récupérer tous les utilisateurs"},
				{"method": "GET", "path": "/users/{id}", "description": "Récupérer un utilisateur par ID"},
				{"method": "POST", "path": "/users", "description": "Créer un utilisateur"},
				{"method": "PUT", "path": "/users/{id}", "description": "Mettre à jour un utilisateur"},
				{"method": "DELETE", "path": "/users/{id}", "description": "Supprimer un utilisateur (soft delete)"},
				{"method": "POST", "path": "/users/{id}/avatar", "description": "Upload avatar utilisateur"},
				{"method": "GET", "path": "/users/{userId}/stats/{period}", "description": "Statistiques utilisateur (daily/weekly/monthly/yearly)"},
				{"method": "GET", "path": "/users/{userId}/charts/{period}", "description": "Données graphiques (week/month/year)"},
				{"method": "GET", "path": "/users/{userId}/workouts", "description": "Sessions d'entraînement d'un utilisateur"},
				{"method": "GET", "path": "/users/{userId}/workouts/stats", "description": "Statistiques d'entraînement"},
				{"method": "GET", "path": "/users/{userId}/workouts/summary", "description": "Résumé des entraînements"},
				{"method": "GET", "path": "/users/{userId}/workouts/records", "description": "Records personnels"},
				{"method": "GET", "path": "/users/{userId}/programs", "description": "Programmes personnalisés d'un utilisateur"},
				{"method": "GET", "path": "/users/{userId}/programs/recommended", "description": "Programmes recommandés"},
				{"method": "GET", "path": "/users/{userId}/challenges/active", "description": "Challenges actifs d'un utilisateur"},
				{"method": "GET", "path": "/users/{userId}/challenges/completed", "description": "Challenges complétés"},
				{"method": "GET", "path": "/users/{userId}/friends/leaderboard", "description": "Classement des amis"},
			},
			"challenges": []map[string]string{
				{"method": "GET", "path": "/challenges", "description": "Récupérer tous les challenges"},
				{"method": "GET", "path": "/challenges/{id}", "description": "Récupérer un challenge par ID"},
				{"method": "POST", "path": "/challenges", "description": "Créer un challenge"},
				{"method": "PUT", "path": "/challenges/{id}", "description": "Mettre à jour un challenge"},
				{"method": "DELETE", "path": "/challenges/{id}", "description": "Supprimer un challenge"},
				{"method": "POST", "path": "/challenges/{id}/like", "description": "Liker un challenge"},
				{"method": "DELETE", "path": "/challenges/{id}/like", "description": "Unliker un challenge"},
				{"method": "POST", "path": "/challenges/{id}/start", "description": "Démarrer un challenge"},
				{"method": "POST", "path": "/challenges/{id}/complete", "description": "Compléter un challenge"},
				{"method": "POST", "path": "/challenges/{id}/tasks/{taskId}", "description": "Compléter une tâche de challenge"},
				{"method": "GET", "path": "/challenges/{id}/progress", "description": "Progression d'un challenge"},
				{"method": "GET", "path": "/challenges/{challengeId}/leaderboard", "description": "Classement d'un challenge"},
			},
			"programs": []map[string]string{
				{"method": "GET", "path": "/programs", "description": "Récupérer tous les programmes"},
				{"method": "GET", "path": "/programs/{id}", "description": "Récupérer un programme par ID"},
				{"method": "POST", "path": "/programs", "description": "Créer un programme"},
				{"method": "PUT", "path": "/programs/{id}", "description": "Mettre à jour un programme"},
				{"method": "DELETE", "path": "/programs/{id}", "description": "Supprimer un programme"},
				{"method": "GET", "path": "/programs/featured", "description": "Programmes en vedette"},
				{"method": "GET", "path": "/programs/popular", "description": "Programmes populaires"},
				{"method": "POST", "path": "/programs/{id}/duplicate", "description": "Dupliquer un programme"},
				{"method": "GET", "path": "/programs/difficulty/{difficulty}", "description": "Programmes par difficulté"},
			},
			"workouts": []map[string]string{
				{"method": "GET", "path": "/workouts", "description": "Récupérer toutes les sessions"},
				{"method": "GET", "path": "/workouts/{id}", "description": "Récupérer une session par ID"},
				{"method": "POST", "path": "/workouts", "description": "Créer une session d'entraînement"},
				{"method": "PATCH", "path": "/workouts/{id}", "description": "Mettre à jour une session"},
				{"method": "DELETE", "path": "/workouts/{id}", "description": "Supprimer une session"},
				{"method": "POST", "path": "/workouts/{sessionId}/sets", "description": "Enregistrer les résultats des séries"},
				{"method": "GET", "path": "/workouts/{sessionId}/sets", "description": "Récupérer les résultats des séries"},
			},
			"leaderboard": []map[string]string{
				{"method": "GET", "path": "/leaderboard", "description": "Classement général (params: period, limit)"},
				{"method": "GET", "path": "/leaderboard/top", "description": "Top 3 performeurs (params: period)"},
				{"method": "GET", "path": "/leaderboard/users/{userId}", "description": "Rang d'un utilisateur (params: period)"},
				{"method": "GET", "path": "/leaderboard/users/{userId}/nearby", "description": "Utilisateurs proches dans le classement"},
			},
			"health": []map[string]string{
				{"method": "GET", "path": "/health", "description": "Health check de l'API"},
			},
		},
		"documentation": map[string]string{
			"description": "API REST pour PumpPro - Application d'entraînement aux pompes",
			"contact":     "support@pompeurpro.com",
		},
	}

	utils.Success(w, routes)
}
