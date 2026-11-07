-- name: SaveSession :exec
INSERT INTO sessions (id, google_subject, expires_at)
VALUES (@id, @google_subject, @expires_at)
ON CONFLICT (id) DO UPDATE
SET google_subject = EXCLUDED.google_subject,
    expires_at = EXCLUDED.expires_at;

-- name: GetSession :one
SELECT id, google_subject, expires_at
FROM sessions
WHERE id = @id;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = @id;
