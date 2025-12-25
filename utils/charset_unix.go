//go:build !windows
// +build !windows

package putils

func SetConsoleCharacterEncodingToUTF8(cp uint32) error {
	//linux placeholder
	//already UTF-8
	return nil
}
