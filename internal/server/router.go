package server

import (
	"net/http"

	"cinekami-server/internal/repos"
	"cinekami-server/internal/routes"
	"cinekami-server/pkg/cache"
	"cinekami-server/pkg/deps"
	"cinekami-server/pkg/signer"
)

type Server struct {
	deps.ServerDeps
}

func New(r *repos.Repository, c cache.Cache, signer signer.Codec) *Server {
	return &Server{ServerDeps: deps.ServerDeps{Repo: r, Cache: c, Signer: signer}}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	sd := s.ServerDeps

	// Endpoints declared here for easy scanning
	mux.HandleFunc("GET /health", routes.Health(sd))
	mux.HandleFunc("GET /movies/active", routes.MoviesActive(sd))
	mux.HandleFunc("GET /movies/{id}/tallies", routes.MovieTallies(sd))
	mux.HandleFunc("POST /votes", routes.Vote(sd))
	mux.HandleFunc("GET /snapshots/{year}/{month}", routes.Snapshots(sd))

	return withCorrelationID(withLogging(mux))
}
