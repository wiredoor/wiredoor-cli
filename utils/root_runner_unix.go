//go:build !windows
// +build !windows

package utils

import (
	"errors"
	"os"
)

func IsRoot() bool {
	if os.Geteuid() != 0 {
		return false
	}
	return true
}
func RelaunchAsRoot() error {

	if os.Geteuid() != 0 {
		return errors.New("Permission denied. This operation requires root privileges.")
	}
	//! verify how to launch with correct permissions
	return nil
}
