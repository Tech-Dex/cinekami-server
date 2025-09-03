package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"cinekami-server/internal/config"
	"cinekami-server/internal/jobs"
	"cinekami-server/internal/migrate"
	"cinekami-server/internal/repos"
	"cinekami-server/internal/server"

	pkgcache "cinekami-server/pkg/cache"
	pkgcrypto "cinekami-server/pkg/crypto"
	pkgdb "cinekami-server/pkg/db"
	pkgtmdb "cinekami-server/pkg/tmdb"
)

func main() {
	_ = godotenv.Load() // best-effort
	cfg := config.FromEnv()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pkgdb.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db connect failed")
	}
	defer pool.Close()

	if err := migrate.Up(cfg.DatabaseURL); err != nil {
		log.Fatal().Err(err).Msg("migrations failed")
	}

	var c pkgcache.Cache
	if addr := cfg.ValkeyAddr; addr != "" {
		vc, err := pkgcache.NewValkey(addr, cfg.ValkeyPassword)
		if err != nil {
			log.Error().Err(err).Msg("valkey connect failed, using in-memory cache")
			c = pkgcache.NewInMemory()
		} else {
			c = vc
		}
	} else {
		c = pkgcache.NewInMemory()
	}

	repository := repos.New(pool)
	signer := pkgcrypto.NewHMAC(cfg.CursorSecret)
	api := server.New(repository, c, signer, cfg.CORSAllowedOrigins)

	// Trigger a one-off test snapshot at startup (temporary for testing).
	// Remove or comment this line after verification.

	// Start background jobs
	var tmdbClient *pkgtmdb.Client
	if cfg.TMDBAPIKey != "" {
		tmdbClient = pkgtmdb.New(cfg.TMDBAPIKey)
	}

	if cfg.TMDBTestMode {
		log.Info().Msg("TMDB test mode enabled; starting fast sync and one-off snapshot")
		jobs.StartTMDBSyncTest(ctx, repository, tmdbClient, cfg.TMDBRegion, cfg.TMDBLanguage)
		jobs.StartTestSnapshot(ctx, repository)
	} else {
		jobs.StartTMDBSync(ctx, repository, tmdbClient, cfg.TMDBRegion, cfg.TMDBLanguage)
	}

	// Seed movies once if table is empty (useful for testing/dev)
	if err := jobs.SeedTMDBIfEmpty(ctx, repository, tmdbClient, cfg.TMDBRegion, cfg.TMDBLanguage); err != nil {
		log.Error().Err(err).Msg("seed from TMDb failed")
	}

	jobs.StartMonthlySnapshot(ctx, repository)

	addr := ":" + cfg.Port
	go func() {
		log.Info().Str("addr", addr).Msg("listening")
		if err := server.StartHTTP(ctx, addr, api.Router()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	<-ctx.Done()
	_, _ = fmt.Fprintln(os.Stderr, "shutting down...")
	time.Sleep(200 * time.Millisecond)
}
