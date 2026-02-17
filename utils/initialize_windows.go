//go:build windows
// +build windows

package utils

import (
	"log"
	"os"

	"golang.org/x/sys/windows/svc"
)

// 1- verify paths existence
// 2- Setup log file locations in case of run as service or as client

// detect if wiredoor folders exists and create missing directories
func verifyFolders() error {
	dir := os.Getenv("PROGRAMDATA") + "\\wiredoor"
	exists := false
	if info, err := os.Lstat(dir); err != nil {
		// :|
	} else {
		exists = info.IsDir()
	}
	if !exists {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// create a log file for clent app or for service
func setLogFileNameAndLocation() error {
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		isSvc = false
	}
	var logFileName string
	if isSvc {
		logFileName = os.Getenv("PROGRAMDATA") + "\\wiredoor\\WiredoorLastServiceLog.txt"
	} else {
		logFileName = os.Getenv("PROGRAMDATA") + "\\wiredoor\\WiredoorLastRunLog.txt"
	}
	logFile, err := os.Create(logFileName)
	if err == nil {
		// defer logFile.Close()
		log.SetOutput(logFile)
	} else {
		//never
		log.Println("Warning: Unable to create log file")
	}
	return nil
}
func setLogFlags() error {

	isSvc, err := svc.IsWindowsService()
	if err != nil {
		isSvc = false
	}
	if isSvc {
		// add file:line
		log.SetFlags(log.Default().Flags() | log.Lshortfile)
	} else {
		flags := log.Default().Flags()
		//remove date
		flags = flags & ^log.Ldate
		//ad file:line
		flags = flags | log.Lshortfile
		log.SetFlags(flags)
	}
	return nil
}
func init() {
	if err := verifyFolders(); err != nil {
		// :|
	}
	if err := setLogFileNameAndLocation(); err != nil {
		// :|
	}
	if err := setLogFlags(); err != nil {
		//:|
	}
}
