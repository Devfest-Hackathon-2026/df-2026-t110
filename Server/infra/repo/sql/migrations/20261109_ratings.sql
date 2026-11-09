BEGIN;

ALTER TABLE users ADD COLUMN rating integer NOT NULL DEFAULT 1000;

-- Earlier matches did not change ratings, so only their outcome is recovered
-- from the stored scores; the rating columns stay NULL.
ALTER TABLE matches ADD COLUMN outcome text;
UPDATE matches
SET outcome = CASE
    WHEN (runs.result -> 'summary' -> 'score' ->> 'player')::bigint
       > (runs.result -> 'summary' -> 'score' ->> 'opponent')::bigint THEN 'win'
    WHEN (runs.result -> 'summary' -> 'score' ->> 'player')::bigint
       < (runs.result -> 'summary' -> 'score' ->> 'opponent')::bigint THEN 'loss'
    ELSE 'draw'
END
FROM runs
WHERE runs.id = matches.run_id;
ALTER TABLE matches ALTER COLUMN outcome SET NOT NULL;
ALTER TABLE matches ADD CHECK (outcome IN ('win', 'loss', 'draw'));

ALTER TABLE matches
    ADD COLUMN opponent_rating integer,
    ADD COLUMN rating_before integer,
    ADD COLUMN rating_after integer,
    ADD CHECK ((opponent_rating IS NULL) = (rating_before IS NULL) AND
               (rating_before IS NULL) = (rating_after IS NULL));

COMMIT;
