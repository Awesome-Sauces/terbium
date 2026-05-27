package log

import (
	"os"

	"github.com/pterm/pterm"
)

var Logger = pterm.DefaultLogger.WithLevel(pterm.LogLevelInfo)

func SetLevel(level pterm.LogLevel) {
	Logger = pterm.DefaultLogger.WithLevel(level)
}

func SetInfo() {
	SetLevel(pterm.LogLevelInfo)
}

func SetDebug() {
	SetLevel(pterm.LogLevelDebug)
}

func SetTrace() {
	SetLevel(pterm.LogLevelTrace)
}

func Trace(message string, args ...any) {
	Logger.Trace(message, Logger.Args(args...))
}

func Debug(message string, args ...any) {
	Logger.Debug(message, Logger.Args(args...))
}

func Info(message string, args ...any) {
	Logger.Info(message, Logger.Args(args...))
}

func Warn(message string, args ...any) {
	Logger.Warn(message, Logger.Args(args...))
}

func Error(message string, args ...any) {
	Logger.Error(message, Logger.Args(args...))
}

func Fatal(err error) {
	Logger.Error(
		"Command failed",
		Logger.Args("error", err),
	)

	os.Exit(1)
}
