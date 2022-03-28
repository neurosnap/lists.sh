package internal

import (
	"os"
	"strings"
)

func GetEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func FsToTitle(basePath string, path string) string {
	return strings.Replace(path, basePath, "", 1)
}
