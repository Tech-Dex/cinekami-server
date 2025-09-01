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
