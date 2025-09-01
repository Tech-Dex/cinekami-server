package routes

import (
	pkgdeps "cinekami-server/pkg/deps"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	pkghttpx "cinekami-server/pkg/httpx"
)

// MovieTallies handles GET /movies/{id}/tallies
func MovieTallies(d pkgdeps.ServerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid id", err))
			return
		}
		cursor := r.URL.Query().Get("signer")
		limitStr := r.URL.Query().Get("limit")
		if limitStr == "" {
			limitStr = "20"
		}
		lim64, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil || lim64 <= 0 || lim64 > 100 {
			pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid limit", err))
			return
		}
		var curCount *int64
		var curCat *string
		if cursor != "" {
			if d.Signer == nil {
				pkghttpx.WriteError(w, r, pkghttpx.Internal("signer signer not configured", nil))
				return
			}
			cnt, cat, decErr := d.Signer.DecodeTalliesCursor(cursor)
			if decErr != nil {
				pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid signer", decErr))
				return
			}
			curCount = &cnt
			curCat = &cat
		}
		cacheKey := "movie_tallies:" + strconv.FormatInt(id, 10) + ":signer:" + cursor + ":limit:" + strconv.FormatInt(lim64, 10)
		if cached, ok := d.Cache.Get(ctx, cacheKey); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(cached))
			return
		}
		items, err := d.Repo.ListTalliesByMoviePage(ctx, id, curCount, curCat, int32(lim64))
		if err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to get tallies", err))
			return
		}
		total, err := d.Repo.CountTalliesByMovie(ctx, id)
		if err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to count tallies", err))
			return
		}
		var next *string
		if len(items) == int(lim64) && d.Signer != nil {
			last := items[len(items)-1]
			nextVal := d.Signer.EncodeTalliesCursor(last.Count, last.Category)
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
