//go:build windows
// +build windows

package utils

import (
	"log"
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

func IsRoot() bool {

	// adminCheck := exec.Command("net", "session")
	// return adminCheck.Run() == nil

	var hToken windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &hToken); err != nil {
		return false
	}
	defer hToken.Close()
	return hToken.IsElevated()
}

func RelaunchAsRoot() error {
	log.Printf("running as Admin ...\n")

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

	// verb := syscall.StringToUTF16Ptr("runas")
	// file := syscall.StringToUTF16Ptr(exe)
	// params := syscall.StringToUTF16Ptr(strings.Join(args, " "))
	// dir := syscall.StringToUTF16Ptr(workDir)
	// show := int32(1) // SW_SHOWNORMAL
	// r, _, err := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW").Call( 0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), uintptr(unsafe.Pointer(params)), uintptr(unsafe.Pointer(dir)), uintptr(show), )
	// // ShellExecute returns >32 on success
	// if r <= 32 {
	// 	return fmt.Errorf("ShellExecuteW failed: %v (code=%d)", err, r)
	//  }

	return err
}
