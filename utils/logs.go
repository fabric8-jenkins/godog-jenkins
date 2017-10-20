package utils

import (
	"fmt"
)

const (
	infoPrefix = "      "
)

// LogInfo info logging
func LogInfo(message string) {
	fmt.Println(infoPrefix + message)
}

// LogInfof info logging
func LogInfof(format string, args ...interface{}) {
	fmt.Printf(infoPrefix + fmt.Sprintf(format, args...))
}
