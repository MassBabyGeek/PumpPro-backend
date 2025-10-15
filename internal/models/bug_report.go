package model

import (
	"encoding/json"
	"time"
)

type BugReport struct {
	ID           string          `json:"id"`
	UserID       *string         `json:"userId,omitempty"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	Category     string          `json:"category"` // bug, crash, ui, feature-request, other
	Severity     string          `json:"severity"` // low, medium, high, critical
	Status       string          `json:"status"`   // open, in-progress, resolved, closed
	DeviceInfo   json.RawMessage `json:"deviceInfo,omitempty"`
	AppVersion   *string         `json:"appVersion,omitempty"`
	PageURL      *string         `json:"pageUrl,omitempty"`
	ErrorStack   *string         `json:"errorStack,omitempty"`
	ScreenshotURL *string        `json:"screenshotUrl,omitempty"`
	UserEmail    *string         `json:"userEmail,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	ResolvedAt   *time.Time      `json:"resolvedAt,omitempty"`
	ResolvedBy   *string         `json:"resolvedBy,omitempty"`
	AdminNotes   *string         `json:"adminNotes,omitempty"`
}

type CreateBugReportRequest struct {
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	Category      string          `json:"category"`
	Severity      string          `json:"severity,omitempty"`
	DeviceInfo    json.RawMessage `json:"deviceInfo,omitempty"`
	AppVersion    string          `json:"appVersion,omitempty"`
	PageURL       string          `json:"pageUrl,omitempty"`
	ErrorStack    string          `json:"errorStack,omitempty"`
	ScreenshotURL string          `json:"screenshotUrl,omitempty"`
	UserEmail     string          `json:"userEmail,omitempty"`
}

type UpdateBugReportRequest struct {
	Status     string `json:"status,omitempty"`
	AdminNotes string `json:"adminNotes,omitempty"`
}
