package repos

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"cinekami-server/internal/model"
	"cinekami-server/internal/store"
)

type TalliesRepo struct {
	db *pgxpool.Pool
	q  *store.Queries
}

func (r *TalliesRepo) GetTallies(ctx context.Context, movieID int64) ([]model.Tally, error) {
	rows, err := r.q.GetTalliesByMovie(ctx, movieID)
	if err != nil {
		return nil, err
	}
	out := make([]model.Tally, 0, len(rows))
	for _, t := range rows {
		cat := categoryToString(t.Category)
		out = append(out, model.Tally{MovieID: t.MovieID, Category: cat, Count: t.Count.Int64})
	}
	return out, nil
}

func (r *TalliesRepo) ListTalliesByMoviePage(ctx context.Context, movieID int64, cursorCount *int64, cursorCategory *string, limit int32) ([]model.Tally, error) {
	var countArg pgtype.Int8
	catArg := ""
	if cursorCount != nil {
		countArg = pgtype.Int8{Int64: *cursorCount, Valid: true}
	}
	if cursorCategory != nil {
		catArg = *cursorCategory
	}
	rows, err := r.q.ListTalliesByMoviePage(ctx, store.ListTalliesByMoviePageParams{
		MovieID: movieID,
		Count:   countArg,
		Column3: catArg,
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]model.Tally, 0, len(rows))
	for _, t := range rows {
		out = append(out, model.Tally{MovieID: t.MovieID, Category: categoryToString(t.Category), Count: t.Count.Int64})
	}
	return out, nil
}

func (r *TalliesRepo) CountTalliesByMovie(ctx context.Context, movieID int64) (int64, error) {
	return r.q.CountTalliesByMovie(ctx, movieID)
}
