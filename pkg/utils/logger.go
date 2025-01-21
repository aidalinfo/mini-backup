package utils

import (
	"fmt"
	"log"
	"os"
)

type Logger struct {
	infoLog  *log.Logger
	errorLog *log.Logger
	debugLog *log.Logger
	file     *os.File
}

// NewLogger creates a new logger that writes to a specified file.
func newLogger(logFilePath string) (*Logger, error) {
	// Open the log file for appending; create it if it doesn't exist.
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &Logger{
		infoLog:  log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLog: log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
		debugLog: log.New(file, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile),
		file:     file,
	}, nil
}

func LoggerFunc() *Logger {
	logger, err := newLogger("logs/backup-tool.log")
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
	}
	return logger
}

// Info logs informational messages with an optional source.
func (l *Logger) Info(msg string, source ...string) {
	message := formatLogMessage(msg, source)
	fmt.Println("🚀 DEV CONSOLE 🚀 -- " + message)
	l.infoLog.Println(message)
}

// Error logs error messages with an optional source.
func (l *Logger) Error(msg string, source ...string) {
	message := formatLogMessage(msg, source)
	fmt.Println("🚀 DEV CONSOLE 🚀 -- " + message)
	l.errorLog.Println(message)
}

// Debug logs debug messages with an optional source.
func (l *Logger) Debug(msg string, source ...string) {
	if GetEnv[string]("LOG_LEVEL") == "debug" {
		message := formatLogMessage(msg, source)
		fmt.Println("🚀 DEV CONSOLE 🚀 -- " + message)
		l.debugLog.Println(message)
	}
}

// Close closes the log file when the logger is no longer needed.
func (l *Logger) Close() error {
	return l.file.Close()
}

// formatLogMessage formats a log message with an optional source.
func formatLogMessage(msg string, source []string) string {
	if len(source) > 0 {
		return fmt.Sprintf("[%s] %s", source[0], msg)
	}
	return msg
}
