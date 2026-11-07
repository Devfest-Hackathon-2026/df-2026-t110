-- Serializes reservations per subject for the rest of the transaction.
-- name: LockUserForGeneration :one
SELECT google_subject
FROM users
WHERE google_subject = @google_subject
FOR UPDATE;

-- name: SumGenerationTokensSince :one
SELECT COALESCE(SUM(tokens), 0)::bigint AS used
FROM generation_usages
WHERE google_subject = @google_subject
  AND reserved_at >= @window_start;

-- name: CreateGenerationUsage :exec
INSERT INTO generation_usages (google_subject, tokens)
VALUES (@google_subject, @tokens);
