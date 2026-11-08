CREATE TABLE IF NOT EXISTS login_attempts (
    id text PRIMARY KEY,
    state text NOT NULL,
    nonce text NOT NULL,
    code_verifier text NOT NULL,
    expires_at timestamptz NOT NULL
);
