package repos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
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

func (r *TalliesRepo) GetVoterCategory(ctx context.Context, movieID int64, fingerprint string) (*string, error) {
	if fingerprint == "" {
		return nil, nil
	}
	cat, err := r.q.GetVoterCategoryByMovieAndFingerprint(ctx, store.GetVoterCategoryByMovieAndFingerprintParams{
		MovieID:     movieID,
		Fingerprint: fingerprint,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &cat, nil
}

func (r *TalliesRepo) GetTalliesAllCategories(ctx context.Context, movieID int64) ([]model.Tally, error) {
	rows, err := r.q.GetTalliesForMovies(ctx, []int64{movieID})
	if err != nil {
		return nil, err
	}
	out := make([]model.Tally, 0, len(rows))
	for _, t := range rows {
		out = append(out, model.Tally{MovieID: anyToInt64(t.MovieID), Category: t.Category, Count: t.Count})
	}
	return out, nil
}
