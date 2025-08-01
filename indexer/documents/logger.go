package documents

import (
	"github.com/aergoio/aergo-lib/log"
)

// Package-level logger for the documents package
var logger *log.Logger

// init function runs automatically when the package is imported
func init() {
	// Initialize with a default logger
	// This will be overridden by SetLogger if called
	logger = log.NewLogger("documents")
}

// SetLogger allows the main application to set a custom logger
// This should be called early in the application startup
func SetLogger(l *log.Logger) {
	logger = l
}

// GetLogger returns the current logger (useful for testing)
func GetLogger() *log.Logger {
	return logger
} 