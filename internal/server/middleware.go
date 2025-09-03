package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/rs/zerolog/log"

	pkgrequestctx "cinekami-server/pkg/requestctx"
)

// StartHTTP starts the HTTP server and blocks until it stops.
func StartHTTP(ctx context.Context, addr string, h http.Handler) error {
	srv := &http.Server{Addr: addr, Handler: h}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("server shutdown error")
		}
	}()
	return srv.ListenAndServe()
}

// correlation id middleware
func withCorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cid := r.Header.Get("X-Correlation-Id")
		if cid == "" {
			cid = xid.New().String()
		}
		// Set on response and ensure request carries it for downstream handlers
		w.Header().Set("X-Correlation-Id", cid)
		r.Header.Set("X-Correlation-Id", cid)
		next.ServeHTTP(w, r.WithContext(pkgrequestctx.WithCorrelationID(r.Context(), cid)))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (sw *statusWriter) WriteHeader(code int) { sw.status = code; sw.ResponseWriter.WriteHeader(code) }
func (sw *statusWriter) Write(b []byte) (int, error) {
	n, err := sw.ResponseWriter.Write(b)
	sw.size += n
	return n, err
}

// logging middleware
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		dur := time.Since(start)
		cid := pkgrequestctx.CorrelationID(r.Context())
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("correlation_id", cid).
			Str("remote_ip", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Int("status", sw.status).
			Int("size", sw.size).
			Dur("duration", dur).
			Msg("http_request")
	})
}

// withCORS adds CORS headers and handles preflight.
func withCORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				allowed := false
				if len(allowedOrigins) == 0 {
					allowed = true // default allow all if not configured
				} else {
					for _, o := range allowedOrigins {
						if o == "*" || strings.EqualFold(o, origin) {
							allowed = true
							break
						}
					}
				}
				if allowed {
					// echo back specific origin if provided and configured, else '*'
					if len(allowedOrigins) == 0 || (len(allowedOrigins) == 1 && allowedOrigins[0] == "*") {
						w.Header().Set("Access-Control-Allow-Origin", "*")
					} else {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Add("Vary", "Origin")
					}
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Fingerprint, X-Correlation-Id")
					w.Header().Set("Access-Control-Expose-Headers", "X-Correlation-Id")
					w.Header().Set("Access-Control-Max-Age", "600")
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// withSecurityHeaders sets common security headers for an API.
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// Minimal CSP for API responses
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		// HSTS (harmless if HTTP, useful if behind TLS)
		w.Header().Set("Strict-Transport-Security", "max-age=15552000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}
