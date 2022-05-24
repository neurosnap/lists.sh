package internal

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gliderlabs/ssh"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type SitePageData struct {
	Domain  template.URL
	HomeURL template.URL
	Email   string
}

var Domain = GetEnv("LISTS_DOMAIN", "lists.sh")
var Email = GetEnv("LISTS_EMAIL", "support@lists.sh")
var SubdomainsEnabled = GetEnv("LISTS_SUBDOMAINS", "0")
var SiteData = SitePageData{
	Domain:  template.URL(Domain),
	HomeURL: template.URL(HomeURL()),
	Email:   Email,
}

func IsSubdomains() bool {
	return SubdomainsEnabled == "1"
}

func BlogURL(username string) string {
	if IsSubdomains() {
		return fmt.Sprintf("//%s.%s", username, Domain)
	}

	return fmt.Sprintf("/%s", username)
}

func RssBlogURL(username string) string {
	if IsSubdomains() {
		return fmt.Sprintf("//%s.%s/rss", username, Domain)
	}

	return fmt.Sprintf("/%s/rss", username)
}

func HomeURL() string {
	if IsSubdomains() {
		return fmt.Sprintf("//%s", Domain)
	}

	return "/"
}

func PostURL(username string, filename string) string {
	fname := url.PathEscape(filename)
	if IsSubdomains() {
		return fmt.Sprintf("//%s.%s/%s", username, Domain, fname)
	}

	return fmt.Sprintf("/%s/%s", username, fname)
}

func ReadURL() string {
	if IsSubdomains() {
		return fmt.Sprintf("https://%s/read", Domain)
	}

	return "/read"
}

func CreateLogger() *zap.SugaredLogger {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	return logger.Sugar()
}

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

const solarYearSecs = 31556926

func TimeAgo(t *time.Time) string {
	d := time.Since(*t)
	var metric string
	var amount int
	if d.Seconds() < 60 {
		amount = int(d.Seconds())
		metric = "second"
	} else if d.Minutes() < 60 {
		amount = int(d.Minutes())
		metric = "minute"
	} else if d.Hours() < 24 {
		amount = int(d.Hours())
		metric = "hour"
	} else if d.Seconds() < solarYearSecs {
		amount = int(d.Hours()) / 24
		metric = "day"
	} else {
		amount = int(d.Seconds()) / solarYearSecs
		metric = "year"
	}
	if amount == 1 {
		return fmt.Sprintf("%d %s ago", amount, metric)
	} else {
		return fmt.Sprintf("%d %ss ago", amount, metric)
	}
}
