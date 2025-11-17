package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

var (
	debugLogger *log.Logger
	enabled     bool
)

// Init initializes the logger with the given debug flag.
// If debug is true, logs are written to the specified file.
// If debug is false, logging is disabled.
func Init(debug bool, logFile string) error {
	enabled = debug
	if !enabled {
		debugLogger = log.New(io.Discard, "", 0)
		return nil
	}

	var writer io.Writer
	if logFile == "" {
		logFile = "/tmp/yescode-tui.log"
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	writer = f
	debugLogger = log.New(writer, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
	debugLogger.Println("Debug logging enabled")

	return nil
}

// Debug logs a debug message if debug logging is enabled.
func Debug(format string, args ...interface{}) {
	if enabled && debugLogger != nil {
		debugLogger.Printf(format, args...)
	}
}

// Info logs an informational message if debug logging is enabled.
func Info(format string, args ...interface{}) {
	if enabled && debugLogger != nil {
		debugLogger.Printf("[INFO] "+format, args...)
	}
}

// Error logs an error message if debug logging is enabled.
func Error(format string, args ...interface{}) {
	if enabled && debugLogger != nil {
		debugLogger.Printf("[ERROR] "+format, args...)
	}
}
