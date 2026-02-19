//go:build windows
// +build windows

package utils

import (
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/sys/windows/svc"
)

func init() {

	//logs
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		isSvc = false
		fmt.Printf("Determine execution service context error: %v\n", err)
	}
	var logger *Logger

	version := "0.9-alpha_windows"
	if isSvc {
		logger, err = New(LoggingOptions{
			File:       os.Getenv("PROGRAMDATA") + "\\wiredoor\\WiredoorServiceLog.json",
			Level:      slog.LevelDebug,
			AppName:    "Wiredoor Service",
			AppVersion: version,
			AddSource:  true,
		})
	} else {
		logger, err = New(LoggingOptions{
			File:       os.Getenv("LOCALAPPDATA") + "\\wiredoor\\WiredoorUserLog.json",
			Level:      slog.LevelDebug,
			AppName:    "Wiredoor User App",
			AppVersion: version,
			AddSource:  true,
		})
	}
	//default log to prevent crash
	if err == nil {
		if logger != nil {
			slog.SetDefault(logger.L)
		} else {
			fmt.Printf("nil logger, using stdout")
		}
	} else {
		fmt.Printf("log initialization error %v", err)
	}
}
