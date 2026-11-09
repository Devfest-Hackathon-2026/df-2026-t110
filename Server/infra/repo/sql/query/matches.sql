-- name: CreateMatchRun :exec
INSERT INTO runs (id, google_subject, field_id, result)
VALUES ($1, $2, $3, $4);

-- name: CreateMatch :exec
INSERT INTO matches (id, run_id, opponent_google_subject, opponent_name, selected_layout_id)
VALUES ($1, $2, $3, $4, $5);

-- name: DrawBattleOpponent :many
WITH candidate AS (
    SELECT users.google_subject, users.name
    FROM users
    JOIN battle_programs ON battle_programs.google_subject = users.google_subject
    JOIN programs ON programs.id = battle_programs.program_id
    WHERE users.google_subject <> sqlc.arg(challenger)
      AND users.name IS NOT NULL AND btrim(users.name) <> ''
    GROUP BY users.google_subject
    HAVING count(*) = sqlc.arg(robot_count)::integer
       AND min(battle_programs.robot_index) = 0
       AND max(battle_programs.robot_index) = sqlc.arg(robot_count)::integer - 1
       AND bool_and(programs.is_battle AND programs.google_subject = users.google_subject)
    ORDER BY random()
    LIMIT 1
)
SELECT candidate.google_subject, candidate.name, programs.id, programs.name AS program_name, programs.source
FROM candidate
JOIN battle_programs ON battle_programs.google_subject = candidate.google_subject
JOIN programs ON programs.id = battle_programs.program_id
ORDER BY battle_programs.robot_index;
