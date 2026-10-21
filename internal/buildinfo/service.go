package buildinfo

import "fmt"

// Strip the installed service host option before the application parser.
func WindowsServiceArguments(arguments []string, defaultName string) ([]string, string, error) {
	result := make([]string, 0, len(arguments))
	name := defaultName
	seen := false
	for index := 0; index < len(arguments); index++ {
		if index > 1 && arguments[index] == "--windows-service" {
			if seen || index+1 >= len(arguments) {
				return nil, "", fmt.Errorf("invalid Windows service option")
			}
			seen = true
			index++
			name = arguments[index]
			if name != "AIDEX" {
				return nil, "", fmt.Errorf("unrecognized AIDEX service")
			}
		} else {
			result = append(result, arguments[index])
		}
	}
	return result, name, nil
}
