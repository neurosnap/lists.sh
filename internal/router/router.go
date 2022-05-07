package router

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/neurosnap/lists.sh/internal/db"
	"go.uber.org/zap"
)

type Route struct {
	method  string
	regex   *regexp.Regexp
	handler http.HandlerFunc
}

func NewRoute(method, pattern string, handler http.HandlerFunc) Route {
	return Route{
		method,
		regexp.MustCompile("^" + pattern + "$"),
		handler,
	}
}

type ServeFn func(http.ResponseWriter, *http.Request)

func CreateServe(routes []Route, subdomainRoutes []Route, dbpool db.DB, logger *zap.SugaredLogger) ServeFn {
	return func(w http.ResponseWriter, r *http.Request) {
		var allow []string
		curRoutes := routes
		subdomain := GetRequestSubdomain(r)
		if subdomain != "" {
			curRoutes = subdomainRoutes
		}

		for _, route := range curRoutes {
			matches := route.regex.FindStringSubmatch(r.URL.Path)
			if len(matches) > 0 {
				if r.Method != route.method {
					allow = append(allow, route.method)
					continue
				}
				loggerCtx := context.WithValue(r.Context(), ctxLoggerKey{}, logger)
				subdomainCtx := context.WithValue(loggerCtx, ctxSubdomainKey{}, subdomain)
				dbCtx := context.WithValue(subdomainCtx, ctxDBKey{}, dbpool)
				ctx := context.WithValue(dbCtx, ctxKey{}, matches[1:])
				route.handler(w, r.WithContext(ctx))
				return
			}
		}
		if len(allow) > 0 {
			w.Header().Set("Allow", strings.Join(allow, ", "))
			http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
			return
		}
		http.NotFound(w, r)
	}
}

type ctxDBKey struct{}
type ctxKey struct{}
type ctxLoggerKey struct{}
type ctxSubdomainKey struct{}

func GetLogger(r *http.Request) *zap.SugaredLogger {
	return r.Context().Value(ctxLoggerKey{}).(*zap.SugaredLogger)
}

func GetDB(r *http.Request) db.DB {
	return r.Context().Value(ctxDBKey{}).(db.DB)
}

func GetField(r *http.Request, index int) string {
	fields := r.Context().Value(ctxKey{}).([]string)
	return fields[index]
}

func GetSubdomain(r *http.Request) string {
	return r.Context().Value(ctxSubdomainKey{}).(string)
}

// https://stackoverflow.com/a/66445657/1713216
func GetRequestSubdomain(r *http.Request) string {
	// The Host that the user queried.
	host := r.Host
	host = strings.TrimSpace(host)
	// Figure out if a subdomain exists in the host given.
	hostParts := strings.Split(host, ".")

	lengthOfHostParts := len(hostParts)

	// scenarios
	// A. site.com  -> length : 2
	// B. www.site.com -> length : 3
	// C. www.hello.site.com -> length : 4

	if lengthOfHostParts == 4 {
		// scenario C
		return strings.Join([]string{hostParts[1]}, "")
	}

	// scenario B with a check
	if lengthOfHostParts == 3 {
		subdomain := strings.Join([]string{hostParts[0]}, "")

		if subdomain == "www" {
			return ""
		} else {
			return subdomain
		}
	}

	return "" // scenario A
}
