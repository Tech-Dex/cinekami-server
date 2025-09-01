-- +migrate Up
CREATE INDEX IF NOT EXISTS idx_movies_popularity_id ON movies (popularity DESC, id DESC);

