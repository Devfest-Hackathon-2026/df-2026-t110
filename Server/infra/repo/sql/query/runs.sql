-- name: CreateRun :exec
INSERT INTO runs (id, google_subject, tutorial_id, field_id, program_id)
VALUES ($1, $2, $3, $4, $5);

-- name: SaveRunResult :execrows
UPDATE runs SET result = $2 WHERE id = $1;

-- name: GetRun :one
SELECT id, google_subject, tutorial_id, field_id, program_id, result
FROM runs WHERE id = $1;
