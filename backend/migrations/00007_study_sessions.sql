-- Logged study time, optionally linked to a tracked skill.
CREATE TABLE study_sessions (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_id   uuid REFERENCES skills(id) ON DELETE SET NULL,
    topic      text NOT NULL,
    minutes    integer NOT NULL CHECK (minutes BETWEEN 1 AND 1440),
    studied_on date NOT NULL,
    resource   text NOT NULL DEFAULT '',
    notes      text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX study_sessions_user_id_idx ON study_sessions (user_id);
CREATE INDEX study_sessions_skill_id_idx ON study_sessions (skill_id);
CREATE INDEX study_sessions_user_studied_idx ON study_sessions (user_id, studied_on DESC);
CREATE INDEX study_sessions_user_updated_idx ON study_sessions (user_id, updated_at DESC);
