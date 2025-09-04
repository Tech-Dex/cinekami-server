package repos

import (
	"context"
	"math"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"cinekami-server/internal/model"
	"cinekami-server/internal/store"

	pkgtmdb "cinekami-server/pkg/tmdb"
)

type MoviesRepo struct {
	db *pgxpool.Pool
	q  *store.Queries
}

type ActiveMoviesSortBy string

type ActiveMoviesSortDir string

const (
	SortByPopularity  ActiveMoviesSortBy = "popularity"
	SortByReleaseDate ActiveMoviesSortBy = "release_date"
	SortBySoloFriends ActiveMoviesSortBy = model.CategorySoloFriends
	SortByCouple      ActiveMoviesSortBy = model.CategoryCouple
	SortByStreaming   ActiveMoviesSortBy = model.CategoryStreaming
	SortByArr         ActiveMoviesSortBy = model.CategoryArr

	SortDirDesc ActiveMoviesSortDir = "desc"
	SortDirAsc  ActiveMoviesSortDir = "asc"
)

type ActiveMoviesFilter struct {
	SortBy      ActiveMoviesSortBy
	SortDir     ActiveMoviesSortDir
	MinPop      *float64
	MaxPop      *float64
	CursorKey   *float64
	CursorID    *int64
	Limit       int32
	Fingerprint *string
}

// ListActiveMoviesPageFiltered returns active movies for the current month with filters and sorting.
func (r *MoviesRepo) ListActiveMoviesPageFiltered(ctx context.Context, now time.Time, f ActiveMoviesFilter) ([]model.Movie, float64, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.SortBy == "" {
		f.SortBy = SortByPopularity
	}
	if f.SortDir != SortDirAsc && f.SortDir != SortDirDesc {
		f.SortDir = SortDirDesc
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
		if f.SortDir == SortDirDesc {
			return math.Inf(+1)
		}
		return math.Inf(-1)
	}()
	curID := int64(0)
	if f.CursorID != nil {
		curID = *f.CursorID
	}
	fp := ""
	if f.Fingerprint != nil {
		fp = *f.Fingerprint
	}
	params := store.ListActiveMoviesFilteredPageParams{
		Column1:     pgtype.Timestamptz{Time: now, Valid: true},
		Column2:     minVal,
		Column3:     maxVal,
		Column4:     string(f.SortBy),
		Column5:     string(f.SortDir),
		Column6:     curKey,
		Column7:     curID,
		Limit:       f.Limit,
		Fingerprint: fp,
	}
	rows, err := r.q.ListActiveMoviesFilteredPage(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	out := make([]model.Movie, 0, len(rows))
	var lastKey float64
	for _, rrow := range rows {
		var votedPtr *string
		if s := categoryToString(rrow.VotedCategory); s != "" {
			v := s
			votedPtr = &v
		}
		mv := model.Movie{
			ID:           rrow.ID,
			Title:        rrow.Title,
			ReleaseDate:  rrow.ReleaseDate.Time,
			Overview:     textPtr(rrow.Overview),
			PosterPath:   textPtr(rrow.PosterPath),
			BackdropPath: textPtr(rrow.BackdropPath),
			Popularity:   rrow.Popularity.Float64,
			Tallies: map[string]int64{
				model.CategorySoloFriends: rrow.SoloFriends,
				model.CategoryCouple:      rrow.Couple,
				model.CategoryStreaming:   rrow.Streaming,
				model.CategoryArr:         rrow.Arr,
			},
			VotedCategory: votedPtr,
			ImdbURL:       textPtr(rrow.ImdbUrl),
			CinemagiaURL:  textPtr(rrow.CinemagiaUrl),
		}
		out = append(out, mv)
		lastKey = anyToFloat64(rrow.KeyValue)
	}
	return out, lastKey, nil
}

func (r *MoviesRepo) CountActiveMoviesFiltered(ctx context.Context, now time.Time, minPop, maxPop *float64) (int64, error) {
	minVal := math.Inf(-1)
	if minPop != nil {
		minVal = *minPop
	}
	maxVal := math.Inf(+1)
	if maxPop != nil {
		maxVal = *maxPop
	}
	arg := store.CountActiveMoviesFilteredParams{
		Column1: pgtype.Timestamptz{Time: now, Valid: true},
		Column2: minVal,
		Column3: maxVal,
	}
	return r.q.CountActiveMoviesFiltered(ctx, arg)
}

// UpsertMovies inserts or updates movies by TMDB id. Returns count upserted.
func (r *MoviesRepo) UpsertMovies(ctx context.Context, movies []pkgtmdb.Movie) (int, error) {
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
			ImdbUrl:      pgtype.Text{Valid: false},
			CinemagiaUrl: pgtype.Text{Valid: false},
		}); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// UpsertMoviesFromTMDB upserts movies and fetches external IDs from TMDb client to populate imdb and cinemagia URLs.
func (r *MoviesRepo) UpsertMoviesFromTMDB(ctx context.Context, movies []pkgtmdb.Movie, c *pkgtmdb.Client) (int, error) {
	// concurrency limit to avoid hammering TMDb or DB
	const concurrency = 10
	var count int64
	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, concurrency)

	for _, m := range movies {
		m := m
		select {
		case <-ctx.Done():
			break
		default:
		}
		sem <- struct{}{}
		g.Go(func() error {
			defer func() { <-sem }()

			// fetch external ids if client present
			imdbURL := ""
			cinemagiaURL := ""
			if c != nil {
				if ext, err := c.GetExternalIDs(m.TMDBID); err == nil {
					if ext.ImdbID != "" {
						imdbURL = "https://www.imdb.com/title/" + ext.ImdbID
						// use URL-escaped title for Cinemagia search, wait for Cinemagia to provide a better solution
						cinemagiaURL = "https://www.cinemagia.ro/cauta/?q=" + url.QueryEscape(m.Title)
					}
				}
			}

			if err := r.q.UpsertMovie(ctx, store.UpsertMovieParams{
				ID:           int64(m.TMDBID),
				Title:        m.Title,
				ReleaseDate:  pgtype.Date{Time: m.ReleaseDate, Valid: true},
				Overview:     textVal(m.Overview),
				PosterPath:   textVal(m.PosterPath),
				BackdropPath: textVal(m.BackdropPath),
				Popularity:   pgtype.Float8{Float64: m.Popularity, Valid: true},
				ImdbUrl:      textVal(imdbURL),
				CinemagiaUrl: textVal(cinemagiaURL),
			}); err != nil {
				return err
			}
			atomic.AddInt64(&count, 1)
			return nil
		})
	}

	// wait for remaining goroutines to finish
	if err := g.Wait(); err != nil {
		return int(count), err
	}
	return int(count), nil
}

func (r *MoviesRepo) HasMovies(ctx context.Context) (bool, error) {
	exists, err := r.q.HasAnyMovies(ctx)
	return exists, err
}

func (r *MoviesRepo) CountActiveMovies(ctx context.Context, now time.Time) (int64, error) {
	return r.q.CountActiveMovies(ctx, pgtype.Timestamptz{Time: now, Valid: true})
}

// Existing method retained for backwards compatibility (unused by new route).
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
