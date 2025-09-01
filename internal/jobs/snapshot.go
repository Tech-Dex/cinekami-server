package jobs

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"cinekami-server/internal/repos"
)

// StartMonthlySnapshot runs a snapshot at the end of each month (00:05 UTC on the 1st).
func StartMonthlySnapshot(ctx context.Context, r *repos.Repository) {
	go func() {
		for {
			now := time.Now().UTC()
			// next run at next month 00:05 UTC
			nextMonth := now.AddDate(0, 1, -now.Day()+1)
			next := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 5, 0, 0, time.UTC)
			t := time.NewTimer(time.Until(next))
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
				// Snapshot previous month
				prev := next.AddDate(0, 0, -1)
				if err := r.SnapshotMonth(ctx, prev.Year(), prev.Month()); err != nil {
					log.Error().Err(err).Msg("snapshot job failed")
				} else {
					log.Info().Int("year", prev.Year()).Int("month", int(prev.Month())).Msg("snapshot job completed")
				}
			}
		}
	}()
}
