-- name: SetBattlePrograms :exec
INSERT INTO battle_programs (google_subject, program_1, program_2, program_3)
VALUES ($1, $2, $3, $4)
ON CONFLICT (google_subject) DO UPDATE SET
    program_1 = EXCLUDED.program_1,
    program_2 = EXCLUDED.program_2,
    program_3 = EXCLUDED.program_3;
