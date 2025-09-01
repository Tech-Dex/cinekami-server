package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"cinekami-server/internal/repos"
	pkgdeps "cinekami-server/pkg/deps"
	pkghttpx "cinekami-server/pkg/httpx"
)

// Vote handles POST /votes
func Vote(d pkgdeps.ServerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type voteReq struct {
			MovieID     int64  `json:"movie_id"`
			Category    string `json:"category"`
			Fingerprint string `json:"fingerprint"`
		}
		type voteResp struct {
			Inserted bool   `json:"inserted"`
			Message  string `json:"message"`
		}

		ctx := r.Context()
		var req voteReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			pkghttpx.WriteError(w, r, pkghttpx.BadRequest("invalid json", err))
			return
		}
		if req.MovieID == 0 || req.Category == "" || req.Fingerprint == "" {
			pkghttpx.WriteError(w, r, pkghttpx.BadRequest("missing fields", nil))
			return
		}
		inserted, err := d.Repo.CreateVote(ctx, req.MovieID, req.Category, req.Fingerprint, time.Now().UTC())
		if err != nil {
			if errors.Is(err, repos.ErrVotingClosed) {
				pkghttpx.WriteError(w, r, pkghttpx.Forbidden("voting closed", err))
				return
			}
			if err.Error() == "movie not found" {
				pkghttpx.WriteError(w, r, pkghttpx.NotFound("movie not found", err))
				return
			}
			pkghttpx.WriteError(w, r, pkghttpx.Internal("failed to record vote", err))
			return
		}
		_ = d.Cache.Delete(ctx, "movie_tallies:"+strconv.FormatInt(req.MovieID, 10))
		_ = d.Cache.Delete(ctx, "active_movies:"+time.Now().UTC().Format("2006-01"))
		pkghttpx.WriteJSON(w, http.StatusOK, voteResp{Inserted: inserted, Message: func() string {
			if inserted {
				return "vote recorded"
			}
			return "duplicate ignored"
		}()})
	}
}
