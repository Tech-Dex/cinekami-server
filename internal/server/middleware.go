package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/xid"
	"github.com/rs/zerolog/log"

	pkgrequestctx "cinekami-server/pkg/requestctx"
)

type errorResp struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteJSON is an exported helper for writing JSON responses from external packages.
func WriteJSON(w http.ResponseWriter, status int, v any) { writeJSON(w, status, v) }

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
