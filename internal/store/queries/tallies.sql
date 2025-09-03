-- name: GetTalliesByMovie :many
SELECT movie_id, category, count
FROM vote_tallies
WHERE movie_id = $1;

-- name: IncrementTally :exec
INSERT INTO vote_tallies (movie_id, category, count)
VALUES ($1, $2, 1)
ON CONFLICT (movie_id, category) DO UPDATE SET count = vote_tallies.count + 1;

-- name: ListTalliesByMoviePage :many
SELECT movie_id, category, count
FROM vote_tallies
WHERE movie_id = $1
  AND (
    $3::text = '' OR (count < $2) OR (count = $2 AND category::text > $3)
  )
ORDER BY count DESC, category ASC
LIMIT $4;

-- name: CountTalliesByMovie :one
SELECT COUNT(*) FROM vote_tallies WHERE movie_id = $1;

-- name: GetTalliesForMovies :many
WITH cats AS (
  SELECT unnest(enum_range(NULL::vote_category))::vote_category AS category
), mids AS (
  SELECT unnest($1::bigint[]) AS movie_id
)
SELECT m.movie_id,
       c.category::text AS category,
       COALESCE(t.count, 0) AS count
FROM mids m
CROSS JOIN cats c
LEFT JOIN vote_tallies t ON t.movie_id = m.movie_id AND t.category = c.category
ORDER BY m.movie_id, c.category;

-- name: GetVoterCategoryByMovieAndFingerprint :one
SELECT v.category::text AS category
FROM votes v
JOIN voters vr ON vr.id = v.voter_id
WHERE v.movie_id = $1 AND vr.fingerprint = $2;
