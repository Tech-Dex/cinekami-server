package repos

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"cinekami-server/internal/model"
	"cinekami-server/internal/store"
	"cinekami-server/pkg/tmdb"
)

type MoviesRepo struct {
	db *pgxpool.Pool
	q  *store.Queries
}

func (r *MoviesRepo) ListActiveMoviesPage(ctx context.Context, now time.Time, cursorPop *float64, cursorID *int64, limit int32) ([]model.Movie, error) {
	pop := 0.0
	id := int64(0)
	if cursorPop != nil {
		pop = *cursorPop
	}
	if cursorID != nil {
		id = *cursorID
	}
	params := store.ListActiveMoviesPageParams{
		pgtype.Timestamptz{Time: now, Valid: true},
		pgtype.Float8{Float64: pop, Valid: true},
		id,
		limit,
	}
	rows, err := r.q.ListActiveMoviesPage(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]model.Movie, 0, len(rows))
	for _, m := range rows {
		out = append(out, model.Movie{
			ID:           m.ID,
			Title:        m.Title,
			ReleaseDate:  m.ReleaseDate.Time,
			Overview:     textPtr(m.Overview),
			PosterPath:   textPtr(m.PosterPath),
			BackdropPath: textPtr(m.BackdropPath),
			Popularity:   m.Popularity.Float64,
		})
	}
	return out, nil
}

// UpsertMovies inserts or updates movies by TMDB id. Returns count upserted.
func (r *MoviesRepo) UpsertMovies(ctx context.Context, movies []tmdb.Movie) (int, error) {
	count := 0
	for _, m := range movies {
		if err := r.q.UpsertMovie(ctx, store.UpsertMovieParams{
			ID:           int64(m.TMDBID),
			Title:        m.Title,
			ReleaseDate:  pgtype.Date{Time: m.ReleaseDate, Valid: true},
			Overview:     textVal(m.Overview),
			PosterPath:   textVal(m.PosterPath),
			BackdropPath: textVal(m.BackdropPath),
			Popularity:   pgtype.Float8{Float64: m.Popularity, Valid: true},
		}); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (r *MoviesRepo) HasMovies(ctx context.Context) (bool, error) {
	exists, err := r.q.HasAnyMovies(ctx)
	return exists, err
}

func (r *MoviesRepo) CountActiveMovies(ctx context.Context, now time.Time) (int64, error) {
	return r.q.CountActiveMovies(ctx, pgtype.Timestamptz{Time: now, Valid: true})
}
