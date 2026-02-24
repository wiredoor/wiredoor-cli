//go:build windows
// +build windows

package utils

import (
	"log/slog"
	"os"

	"github.com/wiredoor/wiredoor-cli/version"
	"golang.org/x/sys/windows/svc"
)

/*
	type LogWritter struct {
		// to use as io.Writter
		level string
		l     *slog.Logger
	}

	func (l *LogWritter) Write(p []byte) (n int, err error) {
		if l != nil {

		}
		return len(p), nil
	}

var WarnLogWtitter = LogWritter{level: "warn", l: slog.Default()}
var ErrorLogWritter = LogWritter{level: "error", l: slog.Default()}
var InfoLogWritter = LogWritter{level: "info", l: slog.Default()}
*/
func init() {
	//logs
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		isSvc = false
		Terminal().Printf("Determine execution service context error: %v\n", err)
	}
	var logger *Logger

	if isSvc {
		logger, err = New(LoggingOptions{
			File:       os.Getenv("PROGRAMDATA") + "\\wiredoor\\WiredoorServiceLog.json",
			Level:      slog.LevelDebug,
			AppName:    "Wiredoor Service",
			AppVersion: version.Version,
			AddSource:  true,
		})
	} else {
		logger, err = New(LoggingOptions{
			File:       os.Getenv("LOCALAPPDATA") + "\\wiredoor\\WiredoorUserLog.json",
			Level:      slog.LevelDebug,
			AppName:    "Wiredoor User App",
			AppVersion: version.Version,
			AddSource:  true,
		})
	}
	//default log to prevent crash
	if err == nil {
		if logger != nil {
			slog.SetDefault(logger.L)
			// ErrorLogWritter.l = logger.L
			// WarnLogWtitter.l = logger.L
			// InfoLogWritter.l = logger.L
		} else {
			Terminal().Errorf("nil logger, using stdout")
		}
	} else {
		Terminal().Errorf("log initialization error %v", err)
	}
}
