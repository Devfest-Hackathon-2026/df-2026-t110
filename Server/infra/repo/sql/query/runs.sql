-- name: CreateRun :exec
INSERT INTO runs (id, google_subject, tutorial_id, field_id, program_id, program_revision)
VALUES ($1, $2, $3, $4, $5, NULLIF(sqlc.arg(program_revision)::integer, 0));

-- name: SaveRunResult :execrows
UPDATE runs SET result = $2 WHERE id = $1;

-- name: GetRun :one
SELECT runs.id, runs.google_subject, runs.tutorial_id, runs.field_id, runs.program_id, runs.program_revision, runs.result,
       matches.opponent_google_subject
FROM runs LEFT JOIN matches ON matches.run_id = runs.id
WHERE runs.id = $1;

-- name: CreateRunPrograms :exec
INSERT INTO run_programs (run_id, side, robot_index, program_id, revision)
SELECT sqlc.arg(run_id), unnest(sqlc.arg(sides)::smallint[]),
       unnest(sqlc.arg(robot_indexes)::smallint[]),
       unnest(sqlc.arg(program_ids)::text[]), unnest(sqlc.arg(revisions)::integer[]);
