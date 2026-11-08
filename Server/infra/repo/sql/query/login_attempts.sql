-- name: SaveLoginAttempt :exec
INSERT INTO login_attempts (id, state, nonce, code_verifier, expires_at)
VALUES (@id, @state, @nonce, @code_verifier, @expires_at);

-- name: ConsumeLoginAttempt :one
DELETE FROM login_attempts
WHERE id = @id
RETURNING id, state, nonce, code_verifier, expires_at;

-- name: DeleteExpiredLoginAttempts :exec
DELETE FROM login_attempts
WHERE expires_at <= now();
