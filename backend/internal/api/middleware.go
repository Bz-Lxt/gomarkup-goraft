package api

import (
	"net/http"
	"strings"

	"goraft/internal/clock"
	"goraft/internal/logutil"
)

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := clock.Now()
		next.ServeHTTP(w, r)
		logutil.Debug("http", "method", r.Method, "path", r.URL.Path, "dur_us", clock.Now().Sub(start).Microseconds())
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Client-Id")
			w.Header().Set("Access-Control-Allow-Methods", "GET,PUT,POST,DELETE,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	host := r.Host
	if strings.Contains(origin, host) {
		return true
	}
	if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
		return true
	}
	return false
}
