package utils

import (
	"fmt"
	"runtime"
)

func FileAndLineStr() string {
	fileLine := ""
	if _, file, line, ok := runtime.Caller(1); ok {
		fileLine = fmt.Sprintf("[%s:%d]", file, line)
	}
	return fileLine
}
