package jobs

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"cinekami-server/internal/repos"

	pkgtmdb "cinekami-server/pkg/tmdb"
)

// SeedTMDBIfEmpty populates movies with current-month TMDb releases if the table is empty.
// Intended for testing/dev convenience; no-op if client is nil or movies already exist.
func SeedTMDBIfEmpty(ctx context.Context, r *repos.Repository, c *pkgtmdb.Client, region, language string) error {
	if c == nil {
		return nil
	}
	has, err := r.HasMovies(ctx)
	if err != nil {
		return err
	}
	if has {
		return nil
	}
	// Compute current month window
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)
	// Fetch all pages for the month
	movies, err := c.DiscoverByReleaseWindow(start, end, region, language, 0)
	if err != nil {
		return err
	}
	n, err := r.UpsertMoviesFromTMDB(ctx, movies, c)
	if err != nil {
		return err
	}
	log.Info().Int("count", n).Msg("seeded movies from TMDb as table was empty")
	return nil
}
