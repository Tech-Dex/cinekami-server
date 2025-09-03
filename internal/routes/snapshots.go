package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cinekami-server/internal/deps"
	"cinekami-server/internal/repos"

	pkghttpx "cinekami-server/pkg/httpx"
)

// Snapshots handles GET /snapshots/{year}/{month}
func Snapshots(d deps.ServerDeps) http.HandlerFunc {
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

		// Parse filters
		sortBy := strings.ToLower(r.URL.Query().Get("sort_by"))
		if sortBy == "" {
			sortBy = string(repos.SnapSortByPopularity)
		}
		sortDir := strings.ToLower(r.URL.Query().Get("sort_dir"))
		if sortDir == "" {
			sortDir = string(repos.SnapSortDirDesc)
		}
		var minPop *float64
		if v := r.URL.Query().Get("min_popularity"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				minPop = &f
			} else {
				pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid min_popularity", err))
				return
			}
		}
		var maxPop *float64
		if v := r.URL.Query().Get("max_popularity"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				maxPop = &f
			} else {
				pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid max_popularity", err))
				return
			}
		}
		if minPop != nil && maxPop != nil && *minPop > *maxPop {
			pkghttpx.WriteError(w, r, pkghttpx.BadRequest("min_popularity > max_popularity", nil))
			return
		}

		cursor := r.URL.Query().Get("cursor")
		limitStr := r.URL.Query().Get("limit")
		if limitStr == "" {
			limitStr = "20"
		}
		lim64, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil || lim64 <= 0 || lim64 > 100 {
			pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid limit", err))
			return
		}
		var curKey *float64
		var curID *int64
		if cursor != "" {
			if d.Codec == nil {
				pkghttpx.WriteError(w, r, pkghttpx.Internal("codec crypto not configured", nil))
				return
			}
			k, id, decErr := d.Codec.DecodeSnapshotsCursor(cursor)
			if decErr != nil {
				pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid cursor", decErr))
				return
			}
			curKey = &k
			curID = &id
		}

		cacheKey := strings.Join([]string{
			"snapshots:", mon,
			":sort:", sortBy,
			":dir:", sortDir,
			":min:", func() string {
				if minPop != nil {
					return strconv.FormatFloat(*minPop, 'f', -1, 64)
				}
				return ""
			}(),
			":max:", func() string {
				if maxPop != nil {
					return strconv.FormatFloat(*maxPop, 'f', -1, 64)
				}
				return ""
			}(),
			":cursor:", cursor,
			":limit:", strconv.FormatInt(lim64, 10),
		}, "")
		if cached, ok := d.Cache.Get(ctx, cacheKey); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(cached))
			return
		}

		items, lastKey, err := d.Repo.ListSnapshotsByMonthFiltered(ctx, repos.SnapshotsFilter{
			Month:     mon,
			SortBy:    repos.SnapshotSortBy(sortBy),
			SortDir:   repos.SnapshotSortDir(sortDir),
			MinPop:    minPop,
			MaxPop:    maxPop,
			CursorKey: curKey,
			CursorID:  curID,
			Limit:     int32(lim64),
		})
		if err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to get snapshots", err))
			return
		}
		total, err := d.Repo.CountSnapshotsByMonthFiltered(ctx, mon, minPop, maxPop)
		if err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to count snapshots", err))
			return
		}
		var next *string
		if len(items) == int(lim64) && d.Codec != nil {
			last := items[len(items)-1]
			nextVal := d.Codec.EncodeSnapshotsCursor(lastKey, last.MovieID)
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

// SnapshotsAvailable handles GET /snapshots/available
func SnapshotsAvailable(d deps.ServerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cacheKey := "snapshots:available"
		if cached, ok := d.Cache.Get(ctx, cacheKey); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(cached))
			return
		}
		rows, err := d.Repo.ListAvailableYearMonths(ctx)
		if err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to list available months", err))
			return
		}
		resp := map[string]any{"items": rows}
		b, _ := json.Marshal(resp)
		_ = d.Cache.Set(ctx, cacheKey, string(b), 24*time.Hour)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}
