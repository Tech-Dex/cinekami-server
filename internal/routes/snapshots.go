package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	pkgdeps "cinekami-server/pkg/deps"
	pkghttpx "cinekami-server/pkg/httpx"
)

// Snapshots handles GET /snapshots/{year}/{month}
func Snapshots(d pkgdeps.ServerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		yearStr := r.PathValue("year")
		monthStr := r.PathValue("month")
		year, err1 := strconv.Atoi(yearStr)
		month, err2 := strconv.Atoi(monthStr)
		if err1 != nil || err2 != nil || month < 1 || month > 12 {
			pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid year/month", nil))
			return
		}
		mon := fmt.Sprintf("%04d-%02d", year, month)
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
		var curMovieID *int64
		if cursor != "" {
			if d.Signer == nil {
				pkghttpx.WriteError(w, r, pkghttpx.Internal("signer signer not configured", nil))
				return
			}
			mid, decErr := d.Signer.DecodeSnapshotsCursor(cursor)
			if decErr != nil {
				pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid signer", decErr))
				return
			}
			curMovieID = &mid
		}
		cacheKey := "snapshots:" + mon + ":signer:" + cursor + ":limit:" + strconv.FormatInt(lim64, 10)
		if cached, ok := d.Cache.Get(ctx, cacheKey); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(cached))
			return
		}
		items, err := d.Repo.ListSnapshotsByMonthPage(ctx, mon, curMovieID, int32(lim64))
		if err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to get snapshots", err))
			return
		}
		total, err := d.Repo.CountSnapshotsByMonth(ctx, mon)
		if err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to count snapshots", err))
			return
		}
		var next *string
		if len(items) == int(lim64) && d.Signer != nil {
			last := items[len(items)-1]
			nextVal := d.Signer.EncodeSnapshotsCursor(last.MovieID)
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
		_ = d.Cache.Set(ctx, cacheKey, string(b), 24*time.Hour)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}
