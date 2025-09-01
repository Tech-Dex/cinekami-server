package routes

import (
	"net/http"

	pkgdeps "cinekami-server/pkg/deps"
	pkghttpx "cinekami-server/pkg/httpx"
)

// Health returns a handler that responds with service status.
func Health(_ pkgdeps.ServerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pkghttpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
