package log

import (
	"os"
	"reflect"

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
	Logger.Trace(message, Logger.Args(flattenArgs(args)...))
}

func Debug(message string, args ...any) {
	Logger.Debug(message, Logger.Args(flattenArgs(args)...))
}

func Info(message string, args ...any) {
	Logger.Info(message, Logger.Args(flattenArgs(args)...))
}

func Warn(message string, args ...any) {
	Logger.Warn(message, Logger.Args(flattenArgs(args)...))
}

func Error(message string, args ...any) {
	Logger.Error(message, Logger.Args(flattenArgs(args)...))
}

func Fatal(err error) {
	Logger.Error(
		"Command failed",
		Logger.Args("error", err),
	)

	os.Exit(1)
}

func flattenArgs(args []any) []any {
	out := make([]any, 0, len(args))

	for _, arg := range args {
		v := reflect.ValueOf(arg)

		if !v.IsValid() {
			out = append(out, arg)
			continue
		}

		switch v.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				out = append(out, v.Index(i).Interface())
			}
		default:
			out = append(out, arg)
		}
	}

	return out
}
