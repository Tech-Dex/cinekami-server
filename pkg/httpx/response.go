package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	pkgrequestctx "cinekami-server/pkg/requestctx"
)

// WriteJSON writes a JSON response with the provided status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError standardizes error responses and logs with correlation id.
func WriteError(w http.ResponseWriter, r *http.Request, he *HTTPError) {
	cid := pkgrequestctx.CorrelationID(r.Context())
	if cid != "" {
		w.Header().Set("X-Correlation-Id", cid)
	}
	payload := map[string]any{
		"error": map[string]any{
			"code":           he.Code,
			"message":        he.Message,
			"correlation_id": cid,
		},
	}
	if he.Details != nil {
		payload["error"].(map[string]any)["details"] = he.Details
	}
	status := he.StatusCode
	if status == 0 {
		status = http.StatusInternalServerError
	}
	log.Error().Str("correlation_id", cid).Str("code", he.Code).Err(he.Err).Msg(he.Message)
	WriteJSON(w, status, payload)
}
