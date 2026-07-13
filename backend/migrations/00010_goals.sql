-- Career goals: skills to learn, certifications to earn, projects to ship.
CREATE TABLE goals (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title        text NOT NULL,
    description  text NOT NULL DEFAULT '',
    category     text NOT NULL CHECK (category IN (
        'job_search','skill','certification','project','networking','other')),
    status       text NOT NULL CHECK (status IN (
        'not_started','in_progress','completed','abandoned')),
    priority     integer NOT NULL CHECK (priority BETWEEN 1 AND 3),
    target_date  date,
    completed_on date,
    notes        text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL
);

CREATE INDEX goals_user_id_idx ON goals (user_id);
CREATE INDEX goals_user_status_idx ON goals (user_id, status);
CREATE INDEX goals_user_updated_idx ON goals (user_id, updated_at DESC);
