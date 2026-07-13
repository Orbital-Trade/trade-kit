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
	switch strings.ToLower(os.Getenv("ALPACA_LOG_LEVEL")) {
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

func Debug(format string, args ...interface{}) { log(DEBUG, "DEBUG", format, args...) }
func Info(format string, args ...interface{})  { log(INFO, " INFO", format, args...) }
func Warn(format string, args ...interface{})  { log(WARN, " WARN", format, args...) }
func Error(format string, args ...interface{}) { log(ERROR, "ERROR", format, args...) }

func log(level Level, label, format string, args ...interface{}) {
	if level < current {
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s  %s  %s\n", ts, label, msg)
}
