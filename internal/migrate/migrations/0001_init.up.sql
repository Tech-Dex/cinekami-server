-- +migrate Up

-- Enum for vote categories (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_type t
        JOIN pg_namespace n ON n.oid = t.typnamespace
        WHERE t.typname = 'vote_category'
    ) THEN
        CREATE TYPE vote_category AS ENUM (
            'solo_friends',
            'couple',
            'streaming',
            'arr'
        );
    END IF;
END $$;

-- Movies table (TMDb id as primary key)
CREATE TABLE IF NOT EXISTS movies (
    id             BIGINT PRIMARY KEY, -- TMDb id
    title          TEXT NOT NULL,
    release_date   DATE NOT NULL,
    overview       TEXT,
    poster_path    TEXT,
    backdrop_path  TEXT,
    popularity     DOUBLE PRECISION DEFAULT 0,
    created_at     TIMESTAMPTZ DEFAULT now(),
    updated_at     TIMESTAMPTZ DEFAULT now()
);

-- Registered users
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT UNIQUE,
    password_hash TEXT,
    password_salt TEXT,
    created_at    TIMESTAMPTZ DEFAULT now()
);

-- Voters (always required)
-- A voter may or may not be linked to a user account
CREATE TABLE IF NOT EXISTS voters (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fingerprint   TEXT NOT NULL UNIQUE,
    user_id       UUID UNIQUE REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ DEFAULT now()
);

-- Votes (event log)
CREATE TABLE IF NOT EXISTS votes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    movie_id   BIGINT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    voter_id   UUID NOT NULL REFERENCES voters(id) ON DELETE CASCADE,
    category   vote_category NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT unique_voter_vote UNIQUE (movie_id, voter_id, category)
);

-- Vote tallies (fast counts)
CREATE TABLE IF NOT EXISTS vote_tallies (
    movie_id   BIGINT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    category   vote_category NOT NULL,
    count      BIGINT DEFAULT 0,
    PRIMARY KEY (movie_id, category)
);

-- Snapshots of tallies (archive)
CREATE TABLE IF NOT EXISTS snapshots (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    month      TEXT NOT NULL, -- YYYY-MM
    movie_id   BIGINT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    tallies    JSONB NOT NULL,
    closed_at  TIMESTAMPTZ DEFAULT now(),
    UNIQUE (month, movie_id)
);
