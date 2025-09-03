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
  AND release_date <= (date_trunc('month', $1::timestamptz)::date + interval '1 month - 1 second')
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
  AND release_date <= (date_trunc('month', $1::timestamptz)::date + interval '1 month - 1 second')
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

-- name: ListActiveMoviesFilteredPage :many
WITH base AS (
  SELECT id, title, release_date, overview, poster_path, backdrop_path, popularity
  FROM movies
  WHERE release_date >= date_trunc('month', $1::timestamptz)::date
    AND release_date <= (date_trunc('month', $1::timestamptz)::date + interval '1 month - 1 second')
    AND $1::timestamptz <= (release_date + interval '14 days')
    AND ($2::float8 IS NULL OR popularity >= $2)
    AND ($3::float8 IS NULL OR popularity <= $3)
), t AS (
  SELECT movie_id,
         SUM(CASE WHEN category = 'solo_friends' THEN count ELSE 0 END)::bigint AS solo_friends,
         SUM(CASE WHEN category = 'couple' THEN count ELSE 0 END)::bigint AS couple,
         SUM(CASE WHEN category = 'streaming' THEN count ELSE 0 END)::bigint AS streaming,
         SUM(CASE WHEN category = 'arr' THEN count ELSE 0 END)::bigint AS arr
  FROM vote_tallies
  GROUP BY movie_id
), joined AS (
  SELECT b.*, COALESCE(t.solo_friends,0) AS solo_friends, COALESCE(t.couple,0) AS couple, COALESCE(t.streaming,0) AS streaming, COALESCE(t.arr,0) AS arr
  FROM base b LEFT JOIN t ON t.movie_id = b.id
), keyed AS (
  SELECT *, CASE
      WHEN $4::text = 'popularity' THEN popularity
      WHEN $4::text = 'release_date' THEN extract(epoch from release_date)
      WHEN $4::text = 'solo_friends' THEN solo_friends::double precision
      WHEN $4::text = 'couple' THEN couple::double precision
      WHEN $4::text = 'streaming' THEN streaming::double precision
      WHEN $4::text = 'arr' THEN arr::double precision
      ELSE popularity
    END AS key_value
  FROM joined
), voted AS (
  SELECT j.*, COALESCE(v.category::text, '') AS voted_category
  FROM keyed j
  LEFT JOIN voters vr ON vr.fingerprint = $9
  LEFT JOIN votes v ON v.movie_id = j.id AND v.voter_id = vr.id
), paged AS (
  SELECT * FROM voted
  WHERE (
    $6::float8 IS NULL OR (
      CASE WHEN $5::text = 'desc'
           THEN (key_value < $6 OR (key_value = $6 AND id < $7::bigint))
           ELSE (key_value > $6 OR (key_value = $6 AND id > $7::bigint))
      END
    )
  )
)
SELECT id, title, release_date, overview, poster_path, backdrop_path, popularity,
       solo_friends, couple, streaming, arr, key_value, voted_category
FROM paged p
ORDER BY
  CASE WHEN $5::text = 'desc' THEN key_value END DESC NULLS LAST,
  CASE WHEN $5::text = 'asc'  THEN key_value END ASC  NULLS LAST,
  CASE WHEN $4::text <> 'popularity' THEN popularity END DESC NULLS LAST,
  CASE WHEN $5::text = 'desc' THEN p.id END DESC NULLS LAST,
  CASE WHEN $5::text = 'asc'  THEN p.id END ASC  NULLS LAST
LIMIT $8;

-- name: CountActiveMoviesFiltered :one
SELECT COUNT(*)
FROM movies
WHERE release_date >= date_trunc('month', $1::timestamptz)::date
  AND release_date <= (date_trunc('month', $1::timestamptz)::date + interval '1 month - 1 second')
  AND $1::timestamptz <= (release_date + interval '14 days')
  AND ($2::float8 IS NULL OR popularity >= $2)
  AND ($3::float8 IS NULL OR popularity <= $3);
