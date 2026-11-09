-- name: CreateProgram :exec
INSERT INTO programs (id, google_subject, name, source, is_battle, tutorial_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetProgram :one
SELECT id, google_subject, name, source, revision, is_battle, tutorial_id
FROM programs WHERE id = $1;

-- name: UpdateProgram :one
UPDATE programs SET name=sqlc.arg(name), source=sqlc.arg(source)
WHERE id=sqlc.arg(id) AND google_subject=sqlc.arg(google_subject)
RETURNING id, google_subject, name, source, revision, is_battle, tutorial_id;

-- name: GetByTutorialID :many
SELECT id, google_subject, name, source, revision, is_battle, tutorial_id
FROM programs WHERE tutorial_id = $1 AND google_subject = $2;
