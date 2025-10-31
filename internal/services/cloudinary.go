package services

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/MassBabyGeek/PumpPro-backend/internal/config"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// CloudinaryService handles all Cloudinary operations
type CloudinaryService struct {
	cld *cloudinary.Cloudinary
}

// NewCloudinaryService creates a new Cloudinary service instance
func NewCloudinaryService(cfg *config.Config) (*CloudinaryService, error) {
	if cfg.CloudinaryCloudName == "" || cfg.CloudinaryAPIKey == "" || cfg.CloudinaryAPISecret == "" {
		return nil, fmt.Errorf("cloudinary configuration is missing")
	}

	cld, err := cloudinary.NewFromParams(
		cfg.CloudinaryCloudName,
		cfg.CloudinaryAPIKey,
		cfg.CloudinaryAPISecret,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudinary: %w", err)
	}

	return &CloudinaryService{
		cld: cld,
	}, nil
}

// UploadAvatar uploads an avatar image to Cloudinary
func (s *CloudinaryService) UploadAvatar(ctx context.Context, file multipart.File, userID string, filename string) (string, error) {
	// Définir le public ID (chemin dans Cloudinary)
	publicID := fmt.Sprintf("avatars/%s", userID)
	overwrite := true

	// Upload vers Cloudinary
	uploadResult, err := s.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:       publicID,
		Folder:         "pumppro/avatars",
		Overwrite:      &overwrite,                  // Écraser l'ancien avatar
		ResourceType:   "image",
		Format:         "jpg",                       // Convertir en JPG
		Transformation: "c_fill,g_face,h_500,w_500", // Redimensionner et centrer sur le visage
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to cloudinary: %w", err)
	}

	// Retourner l'URL sécurisée
	return uploadResult.SecureURL, nil
}

// UploadChallengeImage uploads a challenge image to Cloudinary
func (s *CloudinaryService) UploadChallengeImage(ctx context.Context, file multipart.File, challengeID string) (string, error) {
	publicID := fmt.Sprintf("challenges/%s", challengeID)
	overwrite := true

	uploadResult, err := s.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:       publicID,
		Folder:         "pumppro/challenges",
		Overwrite:      &overwrite,
		ResourceType:   "image",
		Transformation: "c_fill,h_800,w_1200", // Format landscape pour les challenges
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload challenge image: %w", err)
	}

	return uploadResult.SecureURL, nil
}

// UploadBugReportScreenshot uploads a bug report screenshot to Cloudinary
func (s *CloudinaryService) UploadBugReportScreenshot(ctx context.Context, file multipart.File, reportID string) (string, error) {
	publicID := fmt.Sprintf("bug_reports/%s", reportID)
	overwrite := true

	uploadResult, err := s.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:     publicID,
		Folder:       "pumppro/bug_reports",
		Overwrite:    &overwrite,
		ResourceType: "image",
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload screenshot: %w", err)
	}

	return uploadResult.SecureURL, nil
}

// DeleteImage deletes an image from Cloudinary by its public ID
func (s *CloudinaryService) DeleteImage(ctx context.Context, publicID string) error {
	_, err := s.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: "image",
	})
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	return nil
}

// GetOptimizedURL returns an optimized URL for an image with transformations
func (s *CloudinaryService) GetOptimizedURL(publicID string, width, height int) string {
	transformation := fmt.Sprintf("c_fill,w_%d,h_%d,q_auto,f_auto", width, height)
	return fmt.Sprintf("https://res.cloudinary.com/%s/image/upload/%s/%s",
		s.cld.Config.Cloud.CloudName,
		transformation,
		publicID,
	)
}
