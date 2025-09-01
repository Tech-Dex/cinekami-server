package routes

import (
	"cinekami-server/internal/repos"
	"cinekami-server/pkg/cache"
)

// CursorSigner lists the cursor methods the handlers rely on.
// server.CursorSigner satisfies this interface.
type CursorSigner interface {
	EncodeMovies(popularity float64, id int64) string
	DecodeMovies(token string) (float64, int64, error)

	EncodeTallies(count int64, category string) string
	DecodeTallies(token string) (int64, string, error)

	EncodeSnapshots(movieID int64) string
	DecodeSnapshots(token string) (int64, error)
}

// Deps holds the dependencies required by the route handlers.
type Deps struct {
	Repo   *repos.Repository
	Cache  cache.Cache
	Signer CursorSigner
}
