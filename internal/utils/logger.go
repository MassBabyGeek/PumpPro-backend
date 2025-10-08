package utils

import (
	"fmt"
	"log"
	"os"
	"time"
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

// LogInfo affiche un message d'information
func LogInfo(format string, v ...interface{}) {
	InfoLogger.Printf(format, v...)
}

// LogError affiche un message d'erreur
func LogError(format string, v ...interface{}) {
	ErrorLogger.Printf(format, v...)
}

// LogDebug affiche un message de debug
func LogDebug(format string, v ...interface{}) {
	DebugLogger.Printf(format, v...)
}

// LogRequest affiche les détails d'une requête HTTP
func LogRequest(method, path, ip string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] %s %s from %s\n", timestamp, method, path, ip)
}
