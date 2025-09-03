-- name: UpsertSnapshot :exec
INSERT INTO snapshots (month, movie_id, tallies)
VALUES ($1, $2, $3)
ON CONFLICT (month, movie_id) DO UPDATE SET
  tallies = EXCLUDED.tallies,
  closed_at = now();

-- name: GetSnapshotsByMonth :many
SELECT id, month, movie_id, tallies, closed_at
FROM snapshots
WHERE month = $1
ORDER BY movie_id ASC;

-- name: GetSnapshot :one
SELECT id, month, movie_id, tallies, closed_at
FROM snapshots
WHERE month = $1 AND movie_id = $2;

-- name: ListSnapshotsByMonthPage :many
SELECT id, month, movie_id, tallies, closed_at
FROM snapshots
WHERE month = $1
  AND ($2::bigint = 0 OR movie_id > $2)
ORDER BY movie_id ASC
LIMIT $3;

-- name: CountSnapshotsByMonth :one
SELECT COUNT(*) FROM snapshots WHERE month = $1;

-- name: ListSnapshotsByMonthFilteredPage :many
WITH s AS (
  SELECT st.month, st.movie_id, st.tallies, st.closed_at, m.popularity, m.title, m.release_date, m.overview, m.poster_path, m.backdrop_path
  FROM snapshots st
  JOIN movies m ON m.id = st.movie_id
  WHERE st.month = $1
    AND ($2::float8 IS NULL OR m.popularity >= $2)
    AND ($3::float8 IS NULL OR m.popularity <= $3)
), exploded AS (
  SELECT month, movie_id, closed_at, popularity, title, release_date, overview, poster_path, backdrop_path,
    COALESCE((tallies ->> 'solo_friends')::bigint, 0) AS solo_friends,
    COALESCE((tallies ->> 'couple')::bigint, 0) AS couple,
    COALESCE((tallies ->> 'streaming')::bigint, 0) AS streaming,
    COALESCE((tallies ->> 'arr')::bigint, 0) AS arr
  FROM s
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
  FROM exploded
), paged AS (
  SELECT * FROM keyed
  WHERE (
    $6::float8 IS NULL OR (
      CASE WHEN $5::text = 'desc' THEN (key_value < $6 OR (key_value = $6 AND movie_id < $7))
           ELSE (key_value > $6 OR (key_value = $6 AND movie_id > $7))
      END
    )
  )
)
SELECT movie_id, month, closed_at, popularity, title, release_date, overview, poster_path, backdrop_path, solo_friends, couple, streaming, arr, key_value
FROM paged
ORDER BY
  CASE WHEN $5::text = 'desc' THEN key_value END DESC NULLS LAST,
  CASE WHEN $5::text = 'asc'  THEN key_value END ASC  NULLS LAST,
  CASE WHEN $4::text <> 'popularity' THEN popularity END DESC NULLS LAST,
  CASE WHEN $5::text = 'desc' THEN movie_id END DESC NULLS LAST,
  CASE WHEN $5::text = 'asc'  THEN movie_id END ASC  NULLS LAST
LIMIT $8;

-- name: CountSnapshotsByMonthFiltered :one
SELECT COUNT(*)
FROM snapshots st
JOIN movies m ON m.id = st.movie_id
WHERE st.month = $1
  AND ($2::float8 IS NULL OR m.popularity >= $2)
  AND ($3::float8 IS NULL OR m.popularity <= $3);

-- name: ListAvailableSnapshotYearMonths :many
SELECT (split_part(month, '-', 1))::int AS year,
       (split_part(month, '-', 2))::int AS month
FROM snapshots
GROUP BY 1, 2
ORDER BY year DESC, month DESC;
