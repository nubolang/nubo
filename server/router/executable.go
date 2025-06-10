package router

import "strings"

// isExecutable checks if the file has an executable extension.
func isExecutable(filePath string) bool {
	// Check if the file has an executable extension.
	return strings.HasSuffix(filePath, ".nubo")
}
