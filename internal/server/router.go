package server

import (
	"net/http"
	"time"

	"cinekami-server/internal/deps"
	"cinekami-server/internal/repos"
	"cinekami-server/internal/routes"

	pkgcache "cinekami-server/pkg/cache"
	pkgcrypto "cinekami-server/pkg/crypto"
)

type Server struct {
	deps.ServerDeps
}

func New(r *repos.Repository, c pkgcache.Cache, signer pkgcrypto.Codec) *Server {
	return &Server{ServerDeps: deps.ServerDeps{Repo: r, Cache: c, Codec: signer, Name: "cinekami-server", StartedAt: time.Now().UTC()}}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	sd := s.ServerDeps

	// Endpoints declared here for easy scanning
	mux.HandleFunc("GET /health", routes.Health(sd))
	mux.HandleFunc("GET /movies/active", routes.MoviesActive(sd))
	mux.HandleFunc("GET /movies/{id}/tallies", routes.MovieTallies(sd))
	mux.HandleFunc("POST /movies/{id}/votes", routes.MovieVote(sd))
	mux.HandleFunc("GET /snapshots/available", routes.SnapshotsAvailable(sd))
	mux.HandleFunc("GET /snapshots/{year}/{month}", routes.Snapshots(sd))

	return withCorrelationID(withLogging(mux))
}
