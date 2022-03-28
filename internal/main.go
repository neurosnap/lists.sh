package internal

import (
	"os"
	"strings"
    "fmt"
)

func GetEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func FsToTitle(basePath string, path string) string {
    base := fmt.Sprintf("%s/", basePath)
	return strings.Replace(path, base, "", 1)
}
