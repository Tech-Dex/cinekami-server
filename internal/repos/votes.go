package repos

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cinekami-server/internal/model"
	"cinekami-server/internal/store"
)

type VotesRepo struct {
	db *pgxpool.Pool
	q  *store.Queries
}

var ErrVotingClosed = errors.New("voting closed")

// CreateVote inserts a vote (by fingerprint) if not already present and increments tallies.
// Returns inserted=true if a new vote was recorded.
func (r *VotesRepo) CreateVote(ctx context.Context, movieID int64, category string, fingerprint string, now time.Time) (bool, error) {
	// Validate movie and openness
	release, err := r.q.GetMovieReleaseDate(ctx, movieID)
	if release.Valid == false && err == nil {
		return false, errors.New("movie not found")
	}
	if err != nil {
		return false, err
	}
	if now.After(release.Time.Add(14 * 24 * time.Hour)) {
		return false, ErrVotingClosed
	}
	if _, ok := model.AllowedCategories[category]; !ok {
		return false, errors.New("invalid category")
	}
	// Ensure voter exists (by fingerprint)
	voterID, err := r.q.GetVoterByFingerprint(ctx, fingerprint)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			voterID, err = r.q.InsertVoter(ctx, fingerprint)
			if err != nil {
				return false, err
			}
		} else {
			return false, err
		}
	}
	// Insert vote (may be duplicate)
	_, err = r.q.InsertVote(ctx, store.InsertVoteParams{
		MovieID:  movieID,
		VoterID:  voterID,
		Category: category, // enum value as string
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil // duplicate
		}
		return false, err
	}
	// Increment tally
	if err := r.q.IncrementTally(ctx, store.IncrementTallyParams{MovieID: movieID, Category: category}); err != nil {
		return false, err
	}
	return true, nil
}
