package utils

import "fmt"

type OSIdentifier string

const (
	Linux   OSIdentifier = "linux"
	Windows OSIdentifier = "windows"
)

func StringToOSIdentifier(identifier string) (OSIdentifier, error) {
	switch identifier {
	case string(Linux):
		return Linux, nil
	case string(Windows):
		return Windows, nil
	default:
		return "", fmt.Errorf("unknown OS identifier '%s'", identifier)
	}
}
