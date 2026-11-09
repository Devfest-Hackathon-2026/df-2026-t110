CREATE TABLE users (
    google_subject text PRIMARY KEY,
    name text UNIQUE
);

CREATE TABLE sessions (
    id text PRIMARY KEY,
    google_subject text NOT NULL REFERENCES users (google_subject) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);

-- Each row is consumed at most once by the OIDC callback; expired rows are
-- swept on every save so that abandoned sign-ins do not accumulate.
CREATE TABLE login_attempts (
    id text PRIMARY KEY,
    state text NOT NULL,
    nonce text NOT NULL,
    code_verifier text NOT NULL,
    expires_at timestamptz NOT NULL
);

CREATE INDEX login_attempts_expires_at_idx ON login_attempts (expires_at);

CREATE TABLE programs (
    id text PRIMARY KEY,
    google_subject text NOT NULL REFERENCES users (google_subject) ON DELETE CASCADE,
    name text NOT NULL CHECK (btrim(name) <> ''),
    source jsonb NOT NULL,
    is_battle boolean NOT NULL,
    tutorial_id text,
    CHECK ((is_battle AND tutorial_id IS NULL) OR
           (NOT is_battle AND tutorial_id IS NOT NULL AND btrim(tutorial_id) <> ''))
);

CREATE INDEX programs_google_subject_idx ON programs (google_subject);

CREATE TABLE runs (
    id uuid PRIMARY KEY,
    google_subject text NOT NULL REFERENCES users (google_subject) ON DELETE CASCADE,
    tutorial_id text NOT NULL,
    field_id text NOT NULL,
    program_id text NOT NULL REFERENCES programs (id),
    result jsonb
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

-- Three ordered robot slots; repeated program references are allowed.
CREATE TABLE battle_programs (
    google_subject text PRIMARY KEY REFERENCES users (google_subject) ON DELETE CASCADE,
    program_1 text NOT NULL REFERENCES programs (id),
    program_2 text NOT NULL REFERENCES programs (id),
    program_3 text NOT NULL REFERENCES programs (id)
);
