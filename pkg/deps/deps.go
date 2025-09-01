package deps

import (
	"cinekami-server/internal/repos"
	"cinekami-server/pkg/cache"
	"cinekami-server/pkg/signer"
)

// ServerDeps holds the dependencies required by handlers and server.
type ServerDeps struct {
	Repo   *repos.Repository
	Cache  cache.Cache
	Signer signer.Codec
}
