package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cinekami-server/internal/deps"
	"cinekami-server/internal/model"
	"cinekami-server/internal/repos"

	pkghttpx "cinekami-server/pkg/httpx"
)

// category enforces that only allowed values are accepted in JSON
// It implements json.Unmarshaler.
type category string

// sentinel error used by UnmarshalJSON to signal invalid category
var errInvalidCategory = errors.New("invalid category")

func (c *category) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if _, ok := model.AllowedCategories[s]; !ok {
		return errInvalidCategory
	}
	*c = category(s)
	return nil
}

func allowedCategoriesList() string {
	keys := make([]string, 0, len(model.AllowedCategories))
	for k := range model.AllowedCategories {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

// MovieVote handles POST /movies/{id}/votes
func MovieVote(d deps.ServerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type voteReq struct {
			Category    category `json:"category"`
			Fingerprint string   `json:"fingerprint"`
		}
		type voteResp struct {
			Inserted      bool             `json:"inserted"`
			Message       string           `json:"message"`
			Tallies       map[string]int64 `json:"tallies"`
			VotedCategory string           `json:"voted_category"`
		}

		ctx := r.Context()
		idStr := r.PathValue("id")
		ID, err := strconv.ParseInt(idStr, 10, 64)
		var req voteReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if errors.Is(err, errInvalidCategory) {
				pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid category; allowed: "+allowedCategoriesList(), err))
				return
			}
			pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid json", err))
			return
		}
		// Prefer header for fingerprint, fallback to body for compatibility
		fingerprint := r.Header.Get("X-Fingerprint")
		if fingerprint == "" {
			fingerprint = req.Fingerprint
		}
		if ID == 0 || fingerprint == "" { // Category validated by JSON unmarshal
			pkghttpx.WriteError(w, r, pkghttpx.BadRequest("missing fields", nil))
			return
		}
		inserted, err := d.Repo.CreateVote(ctx, ID, string(req.Category), fingerprint, time.Now().UTC())
		if err != nil {
			if errors.Is(err, repos.ErrVotingClosed) {
				pkghttpx.WriteError(w, r, pkghttpx.Forbidden("voting closed", err))
				return
			}
			if err.Error() == "movie not found" {
				pkghttpx.WriteError(w, r, pkghttpx.NotFound("movie not found", err))
				return
			}
			if err.Error() == "invalid category" {
				pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid category; allowed: "+allowedCategoriesList(), err))
				return
			}
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to record vote", err))
			return
		}
		// Fetch current tallies for this movie (includes zeros)
		rows, terr := d.Repo.GetTalliesAllCategories(ctx, ID)
		if terr != nil {
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to load tallies", terr))
			return
		}
		// Build map for FE convenience
		tallyMap := make(map[string]int64, len(model.AllowedCategories))
		for k := range model.AllowedCategories {
			tallyMap[k] = 0
		}
		for _, t := range rows {
			tallyMap[t.Category] = t.Count
		}
		// Determine current user's category from DB (may differ if duplicate vote)
		voted := ""
		if cat, gerr := d.Repo.GetVoterCategory(ctx, ID, fingerprint); gerr == nil && cat != nil {
			voted = *cat
		}
		// Invalidate caches
		_ = d.Cache.DeletePrefix(ctx, "active_movies:"+time.Now().UTC().Format("2006-01"))
		pkghttpx.WriteJSON(w, http.StatusOK, voteResp{Inserted: inserted, Message: func() string {
			if inserted {
				return "vote recorded"
			}
			return "duplicate ignored"
		}(), Tallies: tallyMap, VotedCategory: voted})
	}
}
