BEGIN;

-- Preserve the concrete board chosen for historical battles. Tutorial runs
-- have no matching row and retain their existing variant field IDs.
UPDATE runs
SET field_id = matches.selected_layout_id
FROM matches
WHERE matches.run_id = runs.id;

ALTER TABLE matches DROP COLUMN selected_layout_id;

COMMIT;
