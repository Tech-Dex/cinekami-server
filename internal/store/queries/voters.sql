-- name: GetVoterByFingerprint :one
SELECT id FROM voters WHERE fingerprint = $1;

-- name: InsertVoter :one
INSERT INTO voters (fingerprint)
VALUES ($1)
RETURNING id;

