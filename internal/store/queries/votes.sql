-- name: InsertVote :one
INSERT INTO votes (movie_id, voter_id, category)
VALUES ($1, $2, $3)
ON CONFLICT (movie_id, voter_id, category) DO NOTHING
RETURNING id;

