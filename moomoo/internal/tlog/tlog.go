// Package tlog provides a minimal leveled logger for moomoo-cli.
// All output goes to stderr so stdout stays clean for JSON/table results.
// Level controlled by MOOMOO_LOG_LEVEL env var.
package tlog

import (
	"fmt"
	"os"
	"strings"
	"time"
)

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
	switch strings.ToLower(os.Getenv("MOOMOO_LOG_LEVEL")) {
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

func Debug(format string, args ...interface{}) { emit(DEBUG, "DEBUG", format, args...) }
func Info(format string, args ...interface{})  { emit(INFO, " INFO", format, args...) }
func Warn(format string, args ...interface{})  { emit(WARN, " WARN", format, args...) }
func Error(format string, args ...interface{}) { emit(ERROR, "ERROR", format, args...) }

func emit(level Level, label, format string, args ...interface{}) {
	if level < current {
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(os.Stderr, "%s  %s  %s\n", ts, label, fmt.Sprintf(format, args...))
}
