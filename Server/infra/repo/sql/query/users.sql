-- A CTE combining the insert with a fallback SELECT can return no row when a
-- concurrent transaction inserts the same subject, so the repository runs
-- CreateUserIfAbsent and GetUser separately.

-- name: CreateUserIfAbsent :exec
INSERT INTO users (google_subject, name)
VALUES (@google_subject, sqlc.narg('name'))
ON CONFLICT (google_subject) DO NOTHING;

-- name: GetUser :one
SELECT google_subject, name
FROM users
WHERE google_subject = @google_subject;

-- Returns no row when the user is missing or already named; the caller
-- distinguishes the two with GetUser. A duplicate name fails with a unique
-- violation.
-- name: SetUserNameIfAbsent :one
UPDATE users
SET name = @name
WHERE google_subject = @google_subject
  AND name IS NULL
RETURNING google_subject, name;
