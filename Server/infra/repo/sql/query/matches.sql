-- Ratings are passed as bigint so that PostgreSQL, not a silent int32
-- conversion, rejects values outside the integer columns.
-- name: CreateMatch :exec
INSERT INTO matches (
    id, run_id, opponent_google_subject, opponent_name,
    outcome, opponent_rating, rating_before, rating_after
)
VALUES (
    @id, @run_id, @opponent_google_subject, @opponent_name, @outcome,
    sqlc.arg(opponent_rating)::bigint, sqlc.arg(rating_before)::bigint, sqlc.arg(rating_after)::bigint
);

-- name: GetUserRating :one
SELECT rating
FROM users
WHERE google_subject = @google_subject;

-- Adding to the current value lets the row lock of this UPDATE serialize only the
-- application of concurrent results, without holding a lock during the match.
-- The delta is bigint so that PostgreSQL rejects an out-of-range rating.
-- name: AddUserRating :one
UPDATE users
SET rating = rating + sqlc.arg(delta)::bigint
WHERE google_subject = @google_subject
RETURNING rating;

-- Only users who have challenged at least once and have a display name are ranked.
-- name: ListRanking :many
SELECT
    rank() OVER (ORDER BY users.rating DESC) AS rank,
    users.name::text AS name,
    users.rating,
    count(*) FILTER (WHERE matches.outcome = 'win') AS wins,
    count(*) FILTER (WHERE matches.outcome = 'loss') AS losses,
    count(*) FILTER (WHERE matches.outcome = 'draw') AS draws
FROM users
JOIN runs ON runs.google_subject = users.google_subject
JOIN matches ON matches.run_id = runs.id
WHERE users.name IS NOT NULL AND btrim(users.name) <> ''
GROUP BY users.google_subject
ORDER BY users.rating DESC, users.name;

-- name: ListBattleOpponentCandidates :many
SELECT users.google_subject, users.rating
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
ORDER BY users.google_subject;

-- name: GetBattleTeam :many
SELECT users.google_subject, users.name, users.rating, programs.id, programs.name AS program_name, programs.source, programs.revision
FROM users
JOIN battle_programs ON battle_programs.google_subject = users.google_subject
JOIN programs ON programs.id = battle_programs.program_id
WHERE users.google_subject = @google_subject
ORDER BY battle_programs.robot_index;
