package utils

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	DebugLogger *log.Logger
)

func init() {
	// Initialiser les loggers
	InfoLogger = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
}

// LogInfo affiche un message d'information en jaune
func LogInfo(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	color.Yellow("[INFO] %s", message)
}

// LogError affiche un message d'erreur en rouge
func LogError(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	color.Red("[ERROR] %s", message)
}

// LogDebug affiche un message de debug en cyan (bleu clair)
func LogDebug(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	color.Cyan("[DEBUG] %s", message)
}

// LogRequest affiche les détails d'une requête HTTP en jaune
func LogRequest(method, path, ip string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	color.Yellow("[%s] %s %s from %s", timestamp, method, path, ip)
}
