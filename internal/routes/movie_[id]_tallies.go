package routes

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"

	"cinekami-server/internal/deps"

	pkghttpx "cinekami-server/pkg/httpx"
)

// MovieTallies handles GET /movies/{id}/tallies
func MovieTallies(d deps.ServerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		idStr := r.PathValue("id")
		ID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid ID", err))
			return
		}
		fingerprint := r.Header.Get("X-Fingerprint")

		// Determine voter's selected category (if any)
		var selected string
		if fingerprint != "" {
			if cat, err := d.Repo.GetVoterCategory(ctx, ID, fingerprint); err == nil && cat != nil {
				selected = *cat
			}
		}
		// Fetch all tallies (including zero) for the movie
		tallies, err := d.Repo.GetTalliesAllCategories(ctx, ID)
		if err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to get tallies", err))
			return
		}
		// Sort like before: count desc, category asc
		sort.Slice(tallies, func(i, j int) bool {
			if tallies[i].Count == tallies[j].Count {
				return tallies[i].Category < tallies[j].Category
			}
			return tallies[i].Count > tallies[j].Count
		})
		// Shape response with voter_choice per item
		type item struct {
			MovieID     int64  `json:"movie_id"`
			Category    string `json:"category"`
			Count       int64  `json:"count"`
			VoterChoice bool   `json:"voter_choice"`
		}
		respItems := make([]item, 0, len(tallies))
		for _, t := range tallies {
			respItems = append(respItems, item{
				MovieID:     t.MovieID,
				Category:    t.Category,
				Count:       t.Count,
				VoterChoice: selected != "" && selected == t.Category,
			})
		}
		b, _ := json.Marshal(map[string]any{
			"items": respItems,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}
