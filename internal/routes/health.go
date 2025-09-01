package routes

import (
	"net/http"
)

// Health returns a handler that responds with service status.
func Health(_ Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
