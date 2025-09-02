-- +migrate Up
-- Ensure a voter can only have one vote per movie, regardless of category
ALTER TABLE votes DROP CONSTRAINT IF EXISTS unique_voter_vote;
ALTER TABLE votes ADD CONSTRAINT unique_voter_vote UNIQUE (movie_id, voter_id);

