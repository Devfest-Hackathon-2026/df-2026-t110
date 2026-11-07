CREATE TABLE users (
    google_subject text PRIMARY KEY,
    name text UNIQUE
);

CREATE TABLE sessions (
    id text PRIMARY KEY,
    google_subject text NOT NULL REFERENCES users (google_subject) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);

-- Usage is appended per reservation rather than kept as a per-window counter,
-- so the window length can change without migrating stored counters.
CREATE TABLE generation_usages (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    google_subject text NOT NULL REFERENCES users (google_subject) ON DELETE CASCADE,
    tokens integer NOT NULL CHECK (tokens > 0),
    reserved_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX generation_usages_google_subject_reserved_at_idx
    ON generation_usages (google_subject, reserved_at);
