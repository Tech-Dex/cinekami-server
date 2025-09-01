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
