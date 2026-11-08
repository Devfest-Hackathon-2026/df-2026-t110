CREATE UNIQUE INDEX IF NOT EXISTS users_name_case_insensitive_idx ON users (lower(name)) WHERE name IS NOT NULL;
