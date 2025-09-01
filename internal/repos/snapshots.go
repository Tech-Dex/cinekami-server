package repos

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"cinekami-server/internal/model"
	"cinekami-server/internal/store"
)

type SnapshotsRepo struct {
	db *pgxpool.Pool
	q  *store.Queries
}

func (r *SnapshotsRepo) SnapshotMonth(ctx context.Context, year int, month time.Month) error {
	monStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	ids, err := r.q.ListMovieIDsByMonth(ctx, pgtype.Date{Time: monStart, Valid: true})
	if err != nil {
		return err
	}
	mon := fmt.Sprintf("%04d-%02d", year, int(month))
	for _, id := range ids {
		rows, err := r.q.GetTalliesByMovie(ctx, id)
		if err != nil {
			return err
		}
		m := make(map[string]int64, len(rows))
		for _, t := range rows {
			cat := categoryToString(t.Category)
			m[cat] = t.Count.Int64
		}
		b, _ := json.Marshal(m)
		if err := r.q.UpsertSnapshot(ctx, store.UpsertSnapshotParams{Month: mon, MovieID: id, Tallies: b}); err != nil {
			return err
		}
	}
	return nil
}

func (r *SnapshotsRepo) GetSnapshotsByMonth(ctx context.Context, month string) ([]model.Snapshot, error) {
	rows, err := r.q.GetSnapshotsByMonth(ctx, month)
	if err != nil {
		return nil, err
	}
	out := make([]model.Snapshot, 0, len(rows))
	for _, s := range rows {
		var tallies map[string]int64
		if err := json.Unmarshal(s.Tallies, &tallies); err != nil {
			return nil, fmt.Errorf("decode tallies: %w", err)
		}
		out = append(out, model.Snapshot{Month: s.Month, MovieID: s.MovieID, Tallies: tallies, Closed: s.ClosedAt.Time})
	}
	return out, nil
}

func (r *SnapshotsRepo) ListSnapshotsByMonthPage(ctx context.Context, month string, cursorMovieID *int64, limit int32) ([]model.Snapshot, error) {
	cur := int64(0)
	if cursorMovieID != nil {
		cur = *cursorMovieID
	}
	rows, err := r.q.ListSnapshotsByMonthPage(ctx, store.ListSnapshotsByMonthPageParams{Month: month, Column2: cur, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]model.Snapshot, 0, len(rows))
	for _, s := range rows {
		var tallies map[string]int64
		if err := json.Unmarshal(s.Tallies, &tallies); err != nil {
			return nil, fmt.Errorf("decode tallies: %w", err)
		}
		out = append(out, model.Snapshot{Month: s.Month, MovieID: s.MovieID, Tallies: tallies, Closed: s.ClosedAt.Time})
	}
	return out, nil
}

func (r *SnapshotsRepo) CountSnapshotsByMonth(ctx context.Context, month string) (int64, error) {
	return r.q.CountSnapshotsByMonth(ctx, month)
}
