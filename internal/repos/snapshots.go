package repos

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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
		m := zeroTallies()
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
		// Ensure all categories are present
		zt := zeroTallies()
		mergeTallies(zt, tallies)
		out = append(out, model.Snapshot{Month: s.Month, MovieID: s.MovieID, Tallies: zt, Closed: s.ClosedAt.Time})
	}
	return out, nil
}

type SnapshotSortBy string

type SnapshotSortDir string

const (
	SnapSortByPopularity  SnapshotSortBy = "popularity"
	SnapSortByReleaseDate SnapshotSortBy = "release_date"
	SnapSortBySoloFriends SnapshotSortBy = model.CategorySoloFriends
	SnapSortByCouple      SnapshotSortBy = model.CategoryCouple
	SnapSortByStreaming   SnapshotSortBy = model.CategoryStreaming
	SnapSortByArr         SnapshotSortBy = model.CategoryArr

	SnapSortDirDesc SnapshotSortDir = "desc"
	SnapSortDirAsc  SnapshotSortDir = "asc"
)

type SnapshotsFilter struct {
	Month     string
	SortBy    SnapshotSortBy
	SortDir   SnapshotSortDir
	MinPop    *float64
	MaxPop    *float64
	CursorKey *float64
	CursorID  *int64
	Limit     int32
}

// ListSnapshotsByMonthFiltered returns snapshots for a month with filters and sorting.
func (r *SnapshotsRepo) ListSnapshotsByMonthFiltered(ctx context.Context, f SnapshotsFilter) ([]model.Snapshot, float64, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.SortBy == "" {
		f.SortBy = SnapSortByPopularity
	}
	if f.SortDir != SnapSortDirAsc && f.SortDir != SnapSortDirDesc {
		f.SortDir = SnapSortDirDesc
	}
	minVal := math.Inf(-1)
	if f.MinPop != nil {
		minVal = *f.MinPop
	}
	maxVal := math.Inf(+1)
	if f.MaxPop != nil {
		maxVal = *f.MaxPop
	}
	curKey := func() float64 {
		if f.CursorKey != nil {
			return *f.CursorKey
		}
		if f.SortDir == SnapSortDirDesc {
			return math.Inf(+1)
		}
		return math.Inf(-1)
	}()
	curID := int64(0)
	if f.CursorID != nil {
		curID = *f.CursorID
	}
	params := store.ListSnapshotsByMonthFilteredPageParams{
		Month:   f.Month,
		Column2: minVal,
		Column3: maxVal,
		Column4: string(f.SortBy),
		Column5: string(f.SortDir),
		Column6: curKey,
		MovieID: curID,
		Limit:   f.Limit,
	}
	rows, err := r.q.ListSnapshotsByMonthFilteredPage(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	out := make([]model.Snapshot, 0, len(rows))
	var lastKey float64
	for _, rr := range rows {
		m := zeroTallies()
		m[model.CategorySoloFriends] = anyToInt64(rr.SoloFriends)
		m[model.CategoryCouple] = anyToInt64(rr.Couple)
		m[model.CategoryStreaming] = anyToInt64(rr.Streaming)
		m[model.CategoryArr] = anyToInt64(rr.Arr)
		out = append(out, model.Snapshot{
			Month:        rr.Month,
			MovieID:      rr.MovieID,
			Tallies:      m,
			Closed:       rr.ClosedAt.Time,
			Title:        rr.Title,
			ReleaseDate:  rr.ReleaseDate.Time,
			Overview:     textPtr(rr.Overview),
			PosterPath:   textPtr(rr.PosterPath),
			BackdropPath: textPtr(rr.BackdropPath),
			Popularity:   rr.Popularity.Float64,
		})
		lastKey = anyToFloat64(rr.KeyValue)
	}
	return out, lastKey, nil
}

func (r *SnapshotsRepo) CountSnapshotsByMonthFiltered(ctx context.Context, month string, minPop, maxPop *float64) (int64, error) {
	minVal := math.Inf(-1)
	if minPop != nil {
		minVal = *minPop
	}
	maxVal := math.Inf(+1)
	if maxPop != nil {
		maxVal = *maxPop
	}
	arg := store.CountSnapshotsByMonthFilteredParams{Month: month, Column2: minVal, Column3: maxVal}
	return r.q.CountSnapshotsByMonthFiltered(ctx, arg)
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
		// Ensure all categories are present
		zt := zeroTallies()
		mergeTallies(zt, tallies)
		out = append(out, model.Snapshot{Month: s.Month, MovieID: s.MovieID, Tallies: zt, Closed: s.ClosedAt.Time})
	}
	return out, nil
}

func (r *SnapshotsRepo) CountSnapshotsByMonth(ctx context.Context, month string) (int64, error) {
	return r.q.CountSnapshotsByMonth(ctx, month)
}
