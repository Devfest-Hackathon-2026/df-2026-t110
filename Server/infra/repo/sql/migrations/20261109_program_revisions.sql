BEGIN;
ALTER TABLE programs ADD COLUMN revision integer NOT NULL DEFAULT 1 CHECK (revision > 0);

CREATE TABLE program_revisions (
    program_id text NOT NULL REFERENCES programs(id),
    revision integer NOT NULL CHECK (revision > 0),
    name text NOT NULL,
    source jsonb NOT NULL,
    PRIMARY KEY (program_id, revision)
);

CREATE FUNCTION version_program() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF NEW.id IS DISTINCT FROM OLD.id OR NEW.google_subject IS DISTINCT FROM OLD.google_subject
           OR NEW.is_battle IS DISTINCT FROM OLD.is_battle OR NEW.tutorial_id IS DISTINCT FROM OLD.tutorial_id THEN
            RAISE EXCEPTION 'program identity and purpose are immutable' USING ERRCODE = '23514';
        END IF;
        NEW.revision := OLD.revision + 1;
    ELSE
        NEW.revision := 1;
    END IF;
    RETURN NEW;
END;
$$;
CREATE FUNCTION archive_program() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO program_revisions(program_id, revision, name, source)
    VALUES (NEW.id, NEW.revision, NEW.name, NEW.source);
    RETURN NEW;
END;
$$;
CREATE TRIGGER version_program BEFORE INSERT OR UPDATE ON programs
FOR EACH ROW EXECUTE FUNCTION version_program();
CREATE TRIGGER archive_program AFTER INSERT OR UPDATE ON programs
FOR EACH ROW EXECUTE FUNCTION archive_program();
CREATE FUNCTION protect_program_revision() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'program revisions are immutable' USING ERRCODE = '23514';
END;
$$;
CREATE TRIGGER protect_program_revision BEFORE UPDATE OR DELETE ON program_revisions
FOR EACH ROW EXECUTE FUNCTION protect_program_revision();

INSERT INTO program_revisions SELECT id, revision, name, source FROM programs;
ALTER TABLE runs ADD COLUMN program_revision integer;
ALTER TABLE runs ADD FOREIGN KEY (program_id, program_revision) REFERENCES program_revisions(program_id, revision);

CREATE TABLE run_programs (
    run_id uuid NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    side smallint NOT NULL CHECK (side IN (0,1)),
    robot_index smallint NOT NULL CHECK (robot_index BETWEEN 0 AND 2),
    program_id text NOT NULL,
    revision integer NOT NULL,
    PRIMARY KEY (run_id, side, robot_index),
    FOREIGN KEY (program_id, revision) REFERENCES program_revisions(program_id, revision)
);

COMMIT;
