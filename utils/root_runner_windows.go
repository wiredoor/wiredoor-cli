//go:build windows
// +build windows

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/windows"
)

func IsRoot() bool {
	adminCheck := exec.Command("net", "session")
	return adminCheck.Run() == nil
}

func RelaunchAsRoot() error {
	fmt.Printf("running as Admin ...\n")

	shellCommand := "runas"
	executablePath, _ := os.Executable()
	runnningPath, _ := os.Getwd()

	joinedAppArgs := ""
	{
		appArgs := os.Args[1:]
		joinedAppArgs += "\"" + strings.Join(appArgs, "\" \"") + "\""

	}
	shellCommandPtr := windows.StringToUTF16Ptr(shellCommand)
	executablePathPtr := windows.StringToUTF16Ptr(executablePath)
	runnningPathPtr := windows.StringToUTF16Ptr(runnningPath)
	joinedAppArgsPtr := windows.StringToUTF16Ptr(joinedAppArgs)

	var showCmd int32 = windows.SW_HIDE

	err := windows.ShellExecute(0, shellCommandPtr, executablePathPtr, joinedAppArgsPtr, runnningPathPtr, showCmd)

	return err
}
