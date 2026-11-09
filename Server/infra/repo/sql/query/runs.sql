-- name: CreateRun :exec
INSERT INTO runs (id, google_subject, tutorial_id, field_id, program_id)
VALUES ($1, $2, $3, $4, $5);

-- name: SaveRunResult :execrows
UPDATE runs SET result = $2 WHERE id = $1;

-- name: GetRun :one
SELECT runs.id, runs.google_subject, runs.tutorial_id, runs.field_id, runs.program_id, runs.result,
       matches.opponent_google_subject
FROM runs LEFT JOIN matches ON matches.run_id = runs.id
WHERE runs.id = $1;
