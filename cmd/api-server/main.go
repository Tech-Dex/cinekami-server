package main

import (
	"context"
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
	"cinekami-server/pkg/cache"
	pkgdb "cinekami-server/pkg/db"
	"cinekami-server/pkg/signer"
	"cinekami-server/pkg/tmdb"
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

	var c cache.Cache
	if addr := cfg.ValkeyAddr; addr != "" {
		vc, err := cache.NewValkey(addr, cfg.ValkeyPassword)
		if err != nil {
			log.Error().Err(err).Msg("valkey connect failed, using in-memory cache")
			c = cache.NewInMemory()
		} else {
			c = vc
		}
	} else {
		c = cache.NewInMemory()
	}

	repository := repos.New(pool)
	signer := signer.NewHMAC(cfg.CursorSecret)
	api := server.New(repository, c, signer)

	// Start background jobs
	var tmdbClient *tmdb.Client
	if cfg.TMDBAPIKey != "" {
		tmdbClient = tmdb.New(cfg.TMDBAPIKey)
	}

	// Seed movies once if table is empty (useful for testing/dev)
	if err := jobs.SeedTMDBIfEmpty(ctx, repository, tmdbClient, cfg.TMDBRegion, cfg.TMDBLanguage); err != nil {
		log.Error().Err(err).Msg("seed from TMDb failed")
	}

	jobs.StartTMDBSync(ctx, repository, tmdbClient, cfg.TMDBRegion, cfg.TMDBLanguage)
	jobs.StartMonthlySnapshot(ctx, repository)

	addr := ":" + cfg.Port
	go func() {
		log.Info().Str("addr", addr).Msg("listening")
		if err := server.StartHTTP(ctx, addr, api.Router()); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	<-ctx.Done()
	_, _ = fmt.Fprintln(os.Stderr, "shutting down...")
	time.Sleep(200 * time.Millisecond)
}
