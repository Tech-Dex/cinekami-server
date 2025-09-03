package repos

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"cinekami-server/internal/model"
	"cinekami-server/internal/store"

	pkgtmdb "cinekami-server/pkg/tmdb"
)

type Repository struct {
	db *pgxpool.Pool
	q  *store.Queries

	Movies    *MoviesRepo
	Votes     *VotesRepo
	Tallies   *TalliesRepo
	Snapshots *SnapshotsRepo
}

func New(db *pgxpool.Pool) *Repository {
	q := store.New(db)
	r := &Repository{db: db, q: q}
	r.Movies = &MoviesRepo{db: db, q: q}
	r.Votes = &VotesRepo{db: db, q: q}
	r.Tallies = &TalliesRepo{db: db, q: q}
	r.Snapshots = &SnapshotsRepo{db: db, q: q}
	return r
}

// Forwarders for compatibility
func (r *Repository) ListActiveMoviesPage(ctx context.Context, now time.Time, cursorPop *float64, cursorID *int64, limit int32) ([]model.Movie, error) {
	return r.Movies.ListActiveMoviesPage(ctx, now, cursorPop, cursorID, limit)
}
func (r *Repository) UpsertMovies(ctx context.Context, movies []pkgtmdb.Movie) (int, error) {
	return r.Movies.UpsertMovies(ctx, movies)
}
func (r *Repository) HasMovies(ctx context.Context) (bool, error) { return r.Movies.HasMovies(ctx) }
func (r *Repository) CountActiveMovies(ctx context.Context, now time.Time) (int64, error) {
	return r.Movies.CountActiveMovies(ctx, now)
}

// New filtered/sorted active movies forwarders
func (r *Repository) ListActiveMoviesPageFiltered(ctx context.Context, now time.Time, f ActiveMoviesFilter) ([]model.Movie, float64, error) {
	return r.Movies.ListActiveMoviesPageFiltered(ctx, now, f)
}
func (r *Repository) CountActiveMoviesFiltered(ctx context.Context, now time.Time, minPop, maxPop *float64) (int64, error) {
	return r.Movies.CountActiveMoviesFiltered(ctx, now, minPop, maxPop)
}

func (r *Repository) CreateVote(ctx context.Context, movieID int64, category, fingerprint string, now time.Time) (bool, error) {
	return r.Votes.CreateVote(ctx, movieID, category, fingerprint, now)
}

func (r *Repository) GetTallies(ctx context.Context, movieID int64) ([]model.Tally, error) {
	return r.Tallies.GetTallies(ctx, movieID)
}
func (r *Repository) ListTalliesByMoviePage(ctx context.Context, movieID int64, cursorCount *int64, cursorCategory *string, limit int32) ([]model.Tally, error) {
	return r.Tallies.ListTalliesByMoviePage(ctx, movieID, cursorCount, cursorCategory, limit)
}
func (r *Repository) CountTalliesByMovie(ctx context.Context, movieID int64) (int64, error) {
	return r.Tallies.CountTalliesByMovie(ctx, movieID)
}
func (r *Repository) GetVoterCategory(ctx context.Context, movieID int64, fingerprint string) (*string, error) {
	return r.Tallies.GetVoterCategory(ctx, movieID, fingerprint)
}
func (r *Repository) GetTalliesAllCategories(ctx context.Context, movieID int64) ([]model.Tally, error) {
	return r.Tallies.GetTalliesAllCategories(ctx, movieID)
}

func (r *Repository) SnapshotMonth(ctx context.Context, year int, month time.Month) error {
	return r.Snapshots.SnapshotMonth(ctx, year, month)
}
func (r *Repository) GetSnapshotsByMonth(ctx context.Context, month string) ([]model.Snapshot, error) {
	return r.Snapshots.GetSnapshotsByMonth(ctx, month)
}
func (r *Repository) ListSnapshotsByMonthPage(ctx context.Context, month string, cursorMovieID *int64, limit int32) ([]model.Snapshot, error) {
	return r.Snapshots.ListSnapshotsByMonthPage(ctx, month, cursorMovieID, limit)
}
func (r *Repository) CountSnapshotsByMonth(ctx context.Context, month string) (int64, error) {
	return r.Snapshots.CountSnapshotsByMonth(ctx, month)
}

// New filtered snapshot forwarders
func (r *Repository) ListSnapshotsByMonthFiltered(ctx context.Context, f SnapshotsFilter) ([]model.Snapshot, float64, error) {
	return r.Snapshots.ListSnapshotsByMonthFiltered(ctx, f)
}
func (r *Repository) CountSnapshotsByMonthFiltered(ctx context.Context, month string, minPop, maxPop *float64) (int64, error) {
	return r.Snapshots.CountSnapshotsByMonthFiltered(ctx, month, minPop, maxPop)
}
