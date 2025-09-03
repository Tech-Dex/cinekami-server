package config

import (
	"crypto/rand"
	"log"
	"os"
	"strings"
)

// Config holds runtime configuration loaded from env.
type Config struct {
	Port               string
	DatabaseURL        string
	ValkeyAddr         string
	ValkeyPassword     string
	TMDBAPIKey         string
	TMDBRegion         string
	TMDBLanguage       string
	TMDBTestMode       bool
	Env                string
	CursorSecret       []byte
	CORSAllowedOrigins []string
}

func FromEnv() Config {
	c := Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/cinekami?sslmode=disable"),
		ValkeyAddr:     getEnv("VALKEY_ADDR", "localhost:6379"),
		ValkeyPassword: os.Getenv("VALKEY_PASSWORD"),
		TMDBAPIKey:     os.Getenv("TMDB_API_KEY"),
		TMDBRegion:     getEnv("TMDB_REGION", "RO"),
		TMDBLanguage:   getEnv("TMDB_LANGUAGE", "en-US"),
		TMDBTestMode:   os.Getenv("TMDB_TEST_MODE") == "1",
		Env:            getEnv("ENV", "development"),
	}
	// CORS allowed origins
	if s := os.Getenv("CORS_ALLOWED_ORIGINS"); s != "" {
		parts := strings.Split(s, ",")
		for _, p := range parts {
			if v := strings.TrimSpace(p); v != "" {
				c.CORSAllowedOrigins = append(c.CORSAllowedOrigins, v)
			}
		}
	}
	// crypto secret: optional env CURSOR_SECRET as raw bytes base64 or hex? Keep it raw; if empty, generate ephemeral
	if s := os.Getenv("CURSOR_SECRET"); s != "" {
		c.CursorSecret = []byte(s)
	} else {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err == nil {
			c.CursorSecret = buf
		} else {
			log.Printf("warning: failed to generate crypto secret: %v", err)
			c.CursorSecret = []byte("insecure-default")
		}
	}
	return c
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func MustHave(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env %s", key)
	}
	return v
}
