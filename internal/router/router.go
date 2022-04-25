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

func CreateServe(routes []Route, dbpool db.DB, logger *zap.SugaredLogger) ServeFn {
	return func(w http.ResponseWriter, r *http.Request) {
		var allow []string
		for _, route := range routes {
			matches := route.regex.FindStringSubmatch(r.URL.Path)
			if len(matches) > 0 {
				if r.Method != route.method {
					allow = append(allow, route.method)
					continue
				}
				loggerCtx := context.WithValue(r.Context(), ctxLoggerKey{}, logger)
				dbCtx := context.WithValue(loggerCtx, ctxDBKey{}, dbpool)
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
