package logger

import (
	"fmt"
	"time"
)

// Codes ANSI pour les couleurs
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"
)

// Info log une information générale (bleu)
func Info(message string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s%s%s\n", ColorGray, timestamp, ColorReset, ColorBlue, fmt.Sprintf(message, args...), ColorReset)
}

// Success log un succès (vert)
func Success(message string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s✓ %s%s\n", ColorGray, timestamp, ColorReset, ColorGreen, fmt.Sprintf(message, args...), ColorReset)
}

// Warning log un avertissement (jaune)
func Warning(message string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s⚠ %s%s\n", ColorGray, timestamp, ColorReset, ColorYellow, fmt.Sprintf(message, args...), ColorReset)
}

// Error log une erreur (rouge)
func Error(message string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %s✗ %s%s\n", ColorGray, timestamp, ColorReset, ColorRed, fmt.Sprintf(message, args...), ColorReset)
}

// Request log une requête HTTP avec durée (cyan)
func Request(method, path string, statusCode int, duration time.Duration) {
	timestamp := time.Now().Format("15:04:05")
	var color string
	if statusCode >= 200 && statusCode < 300 {
		color = ColorGreen
	} else if statusCode >= 300 && statusCode < 400 {
		color = ColorCyan
	} else if statusCode >= 400 && statusCode < 500 {
		color = ColorYellow
	} else {
		color = ColorRed
	}

	// Formater la durée
	durationStr := ""
	if duration < time.Millisecond {
		durationStr = fmt.Sprintf("%.0fµs", float64(duration.Microseconds()))
	} else if duration < time.Second {
		durationStr = fmt.Sprintf("%.0fms", float64(duration.Milliseconds()))
	} else {
		durationStr = fmt.Sprintf("%.2fs", duration.Seconds())
	}

	fmt.Printf("%s[%s]%s %s%-6s%s %s%-50s%s %s[%d]%s %s(%s)%s\n",
		ColorGray, timestamp, ColorReset,
		ColorPurple, method, ColorReset,
		ColorWhite, path, ColorReset,
		color, statusCode, ColorReset,
		ColorGray, durationStr, ColorReset)
}

// Debug log un message de debug (gris) - utilisé seulement en développement
func Debug(message string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s] DEBUG: %s%s\n", ColorGray, timestamp, fmt.Sprintf(message, args...), ColorReset)
}
