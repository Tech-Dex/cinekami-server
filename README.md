# CineKami Server

A Go 1.25 server for CineKami: vote on upcoming cinema releases with TMDb sync, PostgreSQL source of truth, Valkey cache, and monthly snapshots.

## Quickstart

- Prereqs: Go 1.25+, PostgreSQL, Valkey (Redis-compatible)
- Copy env: `cp .env.example .env` and adjust values
- Run: `go run ./cmd/api-server`

Optional: start local services via Docker

```bash
# Postgres
docker run --name cinekami-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=cinekami -p 5432:5432 -d postgres:16

# Valkey (Redis-compatible)
docker run --name cinekami-valkey -p 6379:6379 -d valkey/valkey:7
```

## Environment

- `PORT`: HTTP port (default 8080)
- `DATABASE_URL`: e.g. `postgres://postgres:postgres@localhost:5432/cinekami?sslmode=disable`
- `VALKEY_ADDR`: e.g. `localhost:6379`
- `VALKEY_PASSWORD`: set if your Valkey/Redis is password protected
- `TMDB_API_KEY`: used for TMDb sync and seeding
- `TMDB_REGION`: TMDb region (default US)
- `TMDB_LANGUAGE`: TMDb language (default en-US)
- `ENV`: development|production (default development)

## Endpoints

- `GET /health` -> `{"status":"ok"}`
- `GET /movies/active` -> cursor-paginated active movies for current month while voting still open (cached)
  - Query params: `limit` (default 20, max 100), `cursor` format: `<popularity>|<tmdb_id>`
  - Sorted by `popularity DESC, id DESC`
  - Response: `{ "items": [Movie...], "next_cursor": "<popularity>|<tmdb_id>" }` when more pages exist
- `POST /votes` -> body: `{"movie_id":number,"category":"solo_friends|couple|streaming|arr","fingerprint":"opaque"}`
- `GET /movies/{id}/tallies` -> per-category tallies (cached). `id` is the TMDb id.
- `GET /snapshots/{year}/{month}` -> monthly snapshots for `YYYY-MM` (cached)

## Data model (current)

- movies: TMDb id as primary key, plus:
  - `title`, `release_date`, `overview`, `poster_path`, `backdrop_path`, `popularity`
- voters: uuid primary key; unique fingerprint; optional user link
- votes: event log with unique `(movie_id, voter_id, category)`
- vote_tallies: fast counts keyed by `(movie_id, category)`
- snapshots: immutable per month (`YYYY-MM`) and movie; tallies stored as JSON map `{category: count}`

## Migrations / Codegen

- Embedded migrations run automatically on startup
- SQL access is generated with `sqlc` from `internal/store/queries/*.sql`
- To re-generate after query changes:

```bash
sqlc generate
```

## Local testing tips

Insert a test movie (TMDb id as the primary key):

```sql
INSERT INTO movies (id, title, release_date, overview, poster_path, backdrop_path, popularity) 
VALUES (123456, 'Example Movie', CURRENT_DATE, 'A test', '/poster.jpg', '/backdrop.jpg', 99.9);
```

Then vote (fingerprint is required):

```bash
curl -X POST http://localhost:8080/votes \
  -H 'Content-Type: application/json' \
  -d '{"movie_id":123456,"category":"couple","fingerprint":"anon_fingerprint_hash"}'
```

List first page of active movies (20 items):

```bash
curl 'http://localhost:8080/movies/active?limit=20'
```

Use next cursor for subsequent page:

```bash
curl 'http://localhost:8080/movies/active?limit=20&cursor=123.45|123456'
```

List tallies:

```bash
curl http://localhost:8080/movies/123456/tallies
```

## Notes

- Valkey is used only for caching; PostgreSQL is the source of truth
- TMDb sync runs weekly; the app seeds current-month movies on startup if the table is empty (with TMDB_API_KEY set)
