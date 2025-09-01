-- name: UpsertMovie :exec
INSERT INTO movies (id, title, release_date, overview, poster_path, backdrop_path, popularity)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO UPDATE SET
  title = EXCLUDED.title,
  release_date = EXCLUDED.release_date,
  overview = EXCLUDED.overview,
  poster_path = EXCLUDED.poster_path,
  backdrop_path = EXCLUDED.backdrop_path,
  popularity = EXCLUDED.popularity,
  updated_at = now();

-- name: ListActiveMoviesPage :many
SELECT id, title, release_date, overview, poster_path, backdrop_path, popularity
FROM movies
WHERE release_date >= date_trunc('month', $1::timestamptz)::date
  AND $1::timestamptz <= (release_date + interval '14 days')
  AND (
    $3::bigint = 0 OR (popularity < $2) OR (popularity = $2 AND id < $3)
  )
ORDER BY popularity DESC, id DESC
LIMIT $4;

-- name: CountActiveMovies :one
SELECT COUNT(*)
FROM movies
WHERE release_date >= date_trunc('month', $1::timestamptz)::date
  AND $1::timestamptz <= (release_date + interval '14 days');

-- name: GetMovieReleaseDate :one
SELECT release_date
FROM movies
WHERE id = $1;

-- name: HasAnyMovies :one
SELECT EXISTS (SELECT 1 FROM movies LIMIT 1) AS exists;

-- name: ListMovieIDsByMonth :many
SELECT id
FROM movies
WHERE release_date >= date_trunc('month', $1::date)
  AND release_date <  (date_trunc('month', $1::date) + interval '1 month')
ORDER BY id;
