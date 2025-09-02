package jobs

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"cinekami-server/internal/repos"

	pkgtmdb "cinekami-server/pkg/tmdb"
)

// StartTMDBSync starts a weekly ticker that triggers the TMDb sync for current month releases.
func StartTMDBSync(ctx context.Context, r *repos.Repository, c *pkgtmdb.Client, region, language string) {
	if c == nil {
		log.Warn().Msg("TMDb client not configured; skipping weekly sync")
		return
	}
	go func() {
		// Align to next Monday 03:00 UTC
		now := time.Now().UTC()
		// find days until next Monday (Weekday 1)
		daysUntilMonday := (int(time.Monday) - int(now.Weekday()) + 7) % 7
		if daysUntilMonday == 0 {
			// if already Monday past 03:00, schedule next week; else today 03:00
			next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, time.UTC)
			if !next.After(now) {
				now = now.AddDate(0, 0, 7)
				daysUntilMonday = (int(time.Monday) - int(now.Weekday()) + 7) % 7
			}
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, time.UTC).AddDate(0, 0, daysUntilMonday)
		if !next.After(now) {
			next = next.AddDate(0, 0, 7)
		}
		t := time.NewTimer(time.Until(next))
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				// Compute current month window in UTC
				cur := time.Now().UTC()
				start := time.Date(cur.Year(), cur.Month(), 1, 0, 0, 0, 0, time.UTC)
				end := start.AddDate(0, 1, -1)
				movies, err := c.DiscoverByReleaseWindow(start, end, region, language, 0) // all pages
				if err != nil {
					log.Error().Err(err).Msg("tmdb discover failed")
				} else {
					if n, e := r.UpsertMovies(ctx, movies); e != nil {
						log.Error().Err(e).Msg("upsert movies failed")
					} else {
						log.Info().Int("count", n).Msg("tmdb weekly sync upserted movies")
					}
				}
				// Schedule next week
				t.Reset(7 * 24 * time.Hour)
			}
		}
	}()
}

// StartTMDBSyncTest starts a fast sync every 30 seconds for testing purposes.
// It performs the same movie discovery and upsert as the weekly sync but with a 30s ticker.
func StartTMDBSyncTest(ctx context.Context, r *repos.Repository, c *pkgtmdb.Client, region, language string) {
	if c == nil {
		log.Warn().Msg("TMDb client not configured; skipping test sync")
		return
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Compute current month window in UTC
				cur := time.Now().UTC()
				start := time.Date(cur.Year(), cur.Month(), 1, 0, 0, 0, 0, time.UTC)
				end := start.AddDate(0, 1, -1)
				movies, err := c.DiscoverByReleaseWindow(start, end, region, language, 0)
				if err != nil {
					log.Error().Err(err).Msg("tmdb test discover failed")
				} else {
					if n, e := r.UpsertMovies(ctx, movies); e != nil {
						log.Error().Err(e).Msg("upsert movies failed (test)")
					} else {
						log.Info().Int("count", n).Msg("tmdb test sync upserted movies")
					}
				}
			}
		}
	}()
}
