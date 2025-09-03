package routes

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cinekami-server/internal/deps"
	"cinekami-server/internal/repos"

	pkghttpx "cinekami-server/pkg/httpx"
)

// MoviesActive registers GET /movies/active
func MoviesActive(d deps.ServerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := time.Now().UTC()

		// Parse filters
		sortBy := strings.ToLower(r.URL.Query().Get("sort_by"))
		if sortBy == "" {
			sortBy = string(repos.SortByPopularity)
		}
		sortDir := strings.ToLower(r.URL.Query().Get("sort_dir"))
		if sortDir == "" {
			sortDir = string(repos.SortDirDesc)
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

		fingerprint := r.Header.Get("X-Fingerprint")

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
			p, id, decErr := d.Codec.DecodeMoviesCursor(cursor)
			if decErr != nil {
				pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid cursor", decErr))
				return
			}
			curKey = &p
			curID = &id
		}

		cacheKey := strings.Join([]string{
			"active_movies:", now.Format("2006-01"),
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
			":fp:", fingerprint,
		}, "")
		if cached, ok := d.Cache.Get(ctx, cacheKey); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(cached))
			return
		}

		f := repos.ActiveMoviesFilter{
			SortBy:    repos.ActiveMoviesSortBy(sortBy),
			SortDir:   repos.ActiveMoviesSortDir(sortDir),
			MinPop:    minPop,
			MaxPop:    maxPop,
			CursorKey: curKey,
			CursorID:  curID,
			Limit:     int32(lim64),
		}
		if fingerprint != "" {
			f.Fingerprint = &fingerprint
		}
		items, lastKey, err := d.Repo.ListActiveMoviesPageFiltered(ctx, now, f)
		if err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to list active movies", err))
			return
		}
		total, err := d.Repo.CountActiveMoviesFiltered(ctx, now, minPop, maxPop)
		if err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to count active movies", err))
			return
		}
		var next *string
		if len(items) == int(lim64) && d.Codec != nil {
			last := items[len(items)-1]
			nextVal := d.Codec.EncodeMoviesCursor(lastKey, last.ID)
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
