BEGIN;

ALTER TABLE runs ALTER COLUMN tutorial_id DROP NOT NULL;
ALTER TABLE runs ALTER COLUMN program_id DROP NOT NULL;
ALTER TABLE runs ADD CONSTRAINT runs_tutorial_program_pair
    CHECK ((tutorial_id IS NULL) = (program_id IS NULL));

CREATE TABLE matches (
    id uuid PRIMARY KEY,
    run_id uuid NOT NULL UNIQUE REFERENCES runs (id) ON DELETE CASCADE,
    opponent_google_subject text NOT NULL REFERENCES users (google_subject),
    opponent_name text NOT NULL,
    selected_layout_id text NOT NULL
);

COMMIT;
