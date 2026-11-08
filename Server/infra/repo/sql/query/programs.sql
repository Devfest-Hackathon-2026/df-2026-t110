-- name: CreateProgram :exec
INSERT INTO programs (id, google_subject, name, source, is_battle, tutorial_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetProgram :one
SELECT id, google_subject, name, source, is_battle, tutorial_id
FROM programs WHERE id = $1;
