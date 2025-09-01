-- +migrate Up
CREATE INDEX IF NOT EXISTS idx_movies_release_date ON movies (release_date);
CREATE INDEX IF NOT EXISTS idx_votes_movie_category ON votes (movie_id, category);
CREATE INDEX IF NOT EXISTS idx_vote_tallies_movie_category ON vote_tallies (movie_id, category);
CREATE INDEX IF NOT EXISTS idx_snapshots_month_movie ON snapshots (month, movie_id);
