package server

import (
	"net/http"

	"cinekami-server/internal/repos"
	"cinekami-server/internal/routes"
	"cinekami-server/pkg/cache"
)

type Server struct {
	Repo   *repos.Repository
	Cache  cache.Cache
	Signer *CursorSigner
}

func New(r *repos.Repository, c cache.Cache, signer *CursorSigner) *Server {
	return &Server{Repo: r, Cache: c, Signer: signer}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	deps := routes.Deps{Repo: s.Repo, Cache: s.Cache, Signer: s.Signer}

	// Endpoints declared here for easy scanning
	mux.HandleFunc("GET /health", routes.Health(deps))
	mux.HandleFunc("GET /movies/active", routes.MoviesActive(deps))
	mux.HandleFunc("GET /movies/{id}/tallies", routes.MovieTallies(deps))
	mux.HandleFunc("POST /votes", routes.Vote(deps))
	mux.HandleFunc("GET /snapshots/{year}/{month}", routes.Snapshots(deps))

	return withCorrelationID(withLogging(mux))
}
