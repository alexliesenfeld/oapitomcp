package goapitomcp

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CORSOptions configures WithCORS.
type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// WithCORS wraps a handler with lightweight CORS handling. It is intentionally
// separate from NewHandler so applications can use their existing CORS stack.
func WithCORS(next http.Handler, opts CORSOptions) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		applyCORSHeaders(w, r, opts)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func applyCORSHeaders(w http.ResponseWriter, r *http.Request, opts CORSOptions) {
	origin := r.Header.Get("Origin")
	if allowedOrigin, vary := allowedCORSOrigin(origin, opts); allowedOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		if vary {
			addVary(w.Header(), "Origin")
		}
	}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(defaultStrings(opts.AllowedMethods, defaultCORSMethods()), ", "))
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(defaultStrings(opts.AllowedHeaders, defaultCORSHeaders()), ", "))
	if len(opts.ExposedHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(opts.ExposedHeaders, ", "))
	}
	if opts.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if opts.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", strconv.FormatInt(int64(opts.MaxAge/time.Second), 10))
	}
}

func allowedCORSOrigin(origin string, opts CORSOptions) (string, bool) {
	allowedOrigins := defaultStrings(opts.AllowedOrigins, []string{"*"})
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			if opts.AllowCredentials && origin != "" {
				return origin, true
			}
			return "*", false
		}
		if allowed == origin && origin != "" {
			return origin, true
		}
	}
	return "", false
}

func defaultCORSMethods() []string {
	return []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions}
}

func defaultCORSHeaders() []string {
	return []string{"Accept", "Authorization", "Content-Type", "Mcp-Session-Id", "MCP-Protocol-Version"}
}

func defaultStrings(values []string, defaults []string) []string {
	if len(values) > 0 {
		return values
	}
	return defaults
}

func addVary(header http.Header, value string) {
	for _, existing := range strings.Split(header.Get("Vary"), ",") {
		if strings.EqualFold(strings.TrimSpace(existing), value) {
			return
		}
	}
	header.Add("Vary", value)
}
