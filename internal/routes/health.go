package routes

import (
	"net/http"
	"time"

	"cinekami-server/internal/deps"

	pkghttpx "cinekami-server/pkg/httpx"
)

// Health returns a handler that responds with service status.
func Health(d deps.ServerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uptime := int64(time.Since(d.StartedAt).Seconds())
		pkghttpx.WriteJSON(w, http.StatusOK, map[string]any{
			"status":         "ok",
			"service":        d.Name,
			"uptime_seconds": uptime,
		})
	}
}
