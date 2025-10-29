package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/MassBabyGeek/PumpPro-backend/internal/config"
)

// GenerateDefaultAvatar génère un avatar par défaut pour un utilisateur
func GenerateDefaultAvatar(userID, userName string) (string, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", err
	}

	// Créer le dossier s'il n'existe pas
	uploadDir := "uploads/avatars"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", err
	}

	// Utiliser l'API DiceBear pour générer un avatar SVG
	// Style: initials (affiche les initiales de l'utilisateur)
	avatarURL := fmt.Sprintf("https://api.dicebear.com/7.x/initials/svg?seed=%s", userName)

	// Télécharger l'image SVG
	resp, err := http.Get(avatarURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download avatar: status %d", resp.StatusCode)
	}

	// Sauvegarder le SVG en tant que fichier
	filename := userID + ".svg"
	filePath := filepath.Join(uploadDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", err
	}

	// Retourner l'URL complète
	return fmt.Sprintf("%s/avatars/%s", cfg.URL, filename), nil
}
