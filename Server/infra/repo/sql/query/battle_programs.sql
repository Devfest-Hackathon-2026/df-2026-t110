-- name: LockBattleProgramOwner :one
SELECT google_subject FROM users WHERE google_subject = $1 FOR UPDATE;

-- name: DeleteBattlePrograms :exec
DELETE FROM battle_programs WHERE google_subject = $1;

-- name: InsertBattlePrograms :exec
INSERT INTO battle_programs (google_subject, program_id, robot_index)
SELECT sqlc.arg(google_subject), program_id, (slot - 1)::smallint
FROM unnest(sqlc.arg(program_ids)::text[]) WITH ORDINALITY AS p(program_id, slot);
