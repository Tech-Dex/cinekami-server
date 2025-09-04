-- +migrate Up

-- Add columns for external URLs
ALTER TABLE movies
  ADD COLUMN IF NOT EXISTS imdb_url TEXT,
  ADD COLUMN IF NOT EXISTS cinemagia_url TEXT;

