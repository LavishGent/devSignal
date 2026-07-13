-- Interview events, each linked to a tracked application.
CREATE TABLE interviews (
    id               uuid PRIMARY KEY,
    user_id          uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    application_id   uuid NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    stage            text NOT NULL CHECK (stage IN (
        'phone_screen','technical','system_design','behavioral','onsite','final')),
    scheduled_at     timestamptz NOT NULL,
    duration_minutes integer NOT NULL DEFAULT 0 CHECK (duration_minutes >= 0 AND duration_minutes <= 1440),
    interviewer      text NOT NULL DEFAULT '',
    location         text NOT NULL DEFAULT '',
    outcome          text NOT NULL CHECK (outcome IN ('pending','passed','failed','cancelled')),
    prep_notes       text NOT NULL DEFAULT '',
    feedback         text NOT NULL DEFAULT '',
    created_at       timestamptz NOT NULL,
    updated_at       timestamptz NOT NULL
);

CREATE INDEX interviews_user_id_idx ON interviews (user_id);
CREATE INDEX interviews_application_id_idx ON interviews (application_id);
CREATE INDEX interviews_user_scheduled_idx ON interviews (user_id, scheduled_at);
CREATE INDEX interviews_user_updated_idx ON interviews (user_id, updated_at DESC);
