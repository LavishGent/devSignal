-- Personal and portfolio projects.
CREATE TABLE projects (
    id               uuid PRIMARY KEY,
    user_id          uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name             text NOT NULL,
    description      text NOT NULL DEFAULT '',
    status           text NOT NULL CHECK (status IN (
        'idea','active','paused','completed','archived')),
    tech_stack       text NOT NULL DEFAULT '',
    repo_url         text NOT NULL DEFAULT '',
    demo_url         text NOT NULL DEFAULT '',
    started_on       date,
    last_activity_on date,
    notes            text NOT NULL DEFAULT '',
    created_at       timestamptz NOT NULL,
    updated_at       timestamptz NOT NULL
);

CREATE INDEX projects_user_id_idx ON projects (user_id);
CREATE INDEX projects_user_status_idx ON projects (user_id, status);
CREATE INDEX projects_user_updated_idx ON projects (user_id, updated_at DESC);
