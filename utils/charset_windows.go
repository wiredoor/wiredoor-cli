//go:build windows
// +build windows

package utils

import (
	"syscall"
)

func SetConsoleCharacterEncodingToUTF8(cp uint32) error {
	kernel32Lib := syscall.NewLazyDLL("kernel32.dll")
	SetConsoleOutputCP := kernel32Lib.NewProc("SetConsoleOutputCP")
	r, _, err := SetConsoleOutputCP.Call(uintptr(cp))
	if r == 0 {
		return err
	}
	SetConsoleCP := kernel32Lib.NewProc("SetConsoleCP")
	r2, _, err2 := SetConsoleCP.Call(uintptr(cp))
	if r2 == 0 {
		return err2
	}
	return nil
}
