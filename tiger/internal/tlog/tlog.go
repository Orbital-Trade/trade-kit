// Package tlog provides a minimal leveled logger for tiger-cli.
//
// All output goes to stderr so stdout stays clean for JSON/table results.
// Log level is controlled by the TIGER_LOG_LEVEL environment variable:
//
//	TIGER_LOG_LEVEL=debug  — show DEBUG, INFO, WARN, ERROR
//	TIGER_LOG_LEVEL=info   — show INFO, WARN, ERROR (default)
//	TIGER_LOG_LEVEL=warn   — show WARN, ERROR
//	TIGER_LOG_LEVEL=error  — show ERROR only
//	TIGER_LOG_LEVEL=off    — suppress all output
//
// Format:
//
//	2006-01-02 15:04:05  INFO  message
//	2006-01-02 15:04:05  WARN  message
//	2006-01-02 15:04:05 ERROR  message
package tlog

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Level represents a log severity level.
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	OFF
)

var current = INFO

func init() {
	switch strings.ToLower(os.Getenv("TIGER_LOG_LEVEL")) {
	case "debug":
		current = DEBUG
	case "warn":
		current = WARN
	case "error":
		current = ERROR
	case "off", "silent":
		current = OFF
	}
}

// Debug logs a debug message (visible only when TIGER_LOG_LEVEL=debug).
func Debug(format string, args ...interface{}) { log(DEBUG, "DEBUG", format, args...) }

// Info logs an informational message.
func Info(format string, args ...interface{}) { log(INFO, " INFO", format, args...) }

// Warn logs a warning message.
func Warn(format string, args ...interface{}) { log(WARN, " WARN", format, args...) }

// Error logs an error message.
func Error(format string, args ...interface{}) { log(ERROR, "ERROR", format, args...) }

func log(level Level, label, format string, args ...interface{}) {
	if level < current {
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s  %s  %s\n", ts, label, msg)
}
