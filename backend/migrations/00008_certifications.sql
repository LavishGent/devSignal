-- Professional certifications earned or pursued alongside the job search.
CREATE TABLE certifications (
    id             uuid PRIMARY KEY,
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name           text NOT NULL,
    issuer         text NOT NULL,
    status         text NOT NULL CHECK (status IN ('planned','in_progress','earned','expired')),
    earned_on      date,
    expires_on     date,
    credential_id  text NOT NULL DEFAULT '',
    credential_url text NOT NULL DEFAULT '',
    notes          text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL,
    CHECK (earned_on IS NULL OR expires_on IS NULL OR expires_on >= earned_on)
);

CREATE INDEX certifications_user_id_idx ON certifications (user_id);
CREATE INDEX certifications_user_status_idx ON certifications (user_id, status);
CREATE INDEX certifications_user_updated_idx ON certifications (user_id, updated_at DESC);
