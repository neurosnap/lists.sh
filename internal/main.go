package internal

import (
	"encoding/base64"
	"fmt"
	"math"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gliderlabs/ssh"
	"golang.org/x/exp/slices"
)

var fnameRe = regexp.MustCompile(`[-_]+`)

func FilenameToTitle(filename string, title string) string {
	if filename != title {
		return title
	}

	pre := fnameRe.ReplaceAllString(title, " ")
	r := []rune(pre)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func SanitizeFileExt(fname string) string {
	return strings.TrimSuffix(fname, filepath.Ext(fname))
}

func KeyText(s ssh.Session) (string, error) {
	if s.PublicKey() == nil {
		return "", fmt.Errorf("Session doesn't have public key")
	}
	kb := base64.StdEncoding.EncodeToString(s.PublicKey().Marshal())
	return fmt.Sprintf("%s %s", s.PublicKey().Type(), kb), nil
}

func GetEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

// IsText reports whether a significant prefix of s looks like correct UTF-8;
// that is, if it is likely that s is human-readable text.
func IsText(s string) bool {
	const max = 1024 // at least utf8.UTFMax
	if len(s) > max {
		s = s[0:max]
	}
	for i, c := range s {
		if i+utf8.UTFMax > len(s) {
			// last char may be incomplete - ignore
			break
		}
		if c == 0xFFFD || c < ' ' && c != '\n' && c != '\t' && c != '\f' {
			// decoding error or control character - not a text file
			return false
		}
	}
	return true
}

var allowedExtensions = []string{".txt"}

// IsTextFile reports whether the file has a known extension indicating
// a text file, or if a significant chunk of the specified file looks like
// correct UTF-8; that is, if it is likely that the file contains human-
// readable text.
func IsTextFile(text string, filename string) bool {
	ext := pathpkg.Ext(filename)
	if !slices.Contains(allowedExtensions, ext) {
		return false
	}

	num := math.Min(float64(len(text)), 1024)
	return IsText(text[0:int(num)])
}
