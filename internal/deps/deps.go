package deps

import (
	"time"

	"cinekami-server/internal/repos"

	pkgcache "cinekami-server/pkg/cache"
	pkgcrypto "cinekami-server/pkg/crypto"
)

// ServerDeps holds the dependencies required by handlers and server.
type ServerDeps struct {
	Repo      *repos.Repository
	Cache     pkgcache.Cache
	Codec     pkgcrypto.Codec
	Name      string
	StartedAt time.Time
}
