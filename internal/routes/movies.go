package routes

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	pkghttpx "cinekami-server/pkg/httpx"
)

// MoviesActive registers GET /movies/active
func MoviesActive(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := time.Now().UTC()
		cursor := r.URL.Query().Get("cursor")
		limitStr := r.URL.Query().Get("limit")
		if limitStr == "" {
			limitStr = "20"
		}
		lim64, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil || lim64 <= 0 || lim64 > 100 {
			writeError(w, r, pkghttpx.BadRequest("invalid limit", err))
			return
		}
		var curPop *float64
		var curID *int64
		if cursor != "" {
			if d.Signer == nil {
				writeError(w, r, pkghttpx.Internal("cursor signer not configured", nil))
				return
			}
			p, id, decErr := d.Signer.DecodeMovies(cursor)
			if decErr != nil {
				writeError(w, r, pkghttpx.BadRequest("invalid cursor", decErr))
				return
			}
			curPop = &p
			curID = &id
		}
		cacheKey := "active_movies:" + now.Format("2006-01") + ":cursor:" + cursor + ":limit:" + strconv.FormatInt(lim64, 10)
		if cached, ok := d.Cache.Get(ctx, cacheKey); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(cached))
			return
		}
		items, err := d.Repo.ListActiveMoviesPage(ctx, now, curPop, curID, int32(lim64))
		if err != nil {
			writeError(w, r, pkghttpx.Internal("failed to list active movies", err))
			return
		}
		total, err := d.Repo.CountActiveMovies(ctx, now)
		if err != nil {
			writeError(w, r, pkghttpx.Internal("failed to count active movies", err))
			return
		}
		var next *string
		if len(items) == int(lim64) && d.Signer != nil {
			last := items[len(items)-1]
			nextVal := d.Signer.EncodeMovies(last.Popularity, last.ID)
			next = &nextVal
		}
		resp := map[string]any{
			"items": items,
			"count": len(items),
			"total": total,
		}
		if next != nil {
			resp["next_cursor"] = *next
		}
		b, _ := json.Marshal(resp)
		_ = d.Cache.Set(ctx, cacheKey, string(b), 2*time.Minute)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}
