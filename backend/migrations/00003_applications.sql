-- Job applications, the core of the job-search pipeline.
CREATE TABLE applications (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company    text NOT NULL,
    position   text NOT NULL,
    status     text NOT NULL CHECK (status IN (
        'wishlist','applied','phone_screen','interviewing',
        'offer','accepted','rejected','withdrawn')),
    applied_on date,
    location   text NOT NULL DEFAULT '',
    work_mode  text NOT NULL DEFAULT '' CHECK (work_mode IN ('', 'remote','hybrid','onsite')),
    url        text NOT NULL DEFAULT '',
    salary_min integer CHECK (salary_min >= 0),
    salary_max integer CHECK (salary_max >= 0),
    notes      text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX applications_user_id_idx ON applications (user_id);
CREATE INDEX applications_user_status_idx ON applications (user_id, status);
CREATE INDEX applications_user_updated_idx ON applications (user_id, updated_at DESC);
