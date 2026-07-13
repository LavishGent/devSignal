-- Professional and recruiter contacts in the job-search network.
CREATE TABLE contacts (
    id                uuid PRIMARY KEY,
    user_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name              text NOT NULL,
    company           text NOT NULL DEFAULT '',
    role              text NOT NULL DEFAULT '',
    relationship      text NOT NULL CHECK (relationship IN (
        'recruiter','hiring_manager','engineer','mentor','peer','other')),
    email             text NOT NULL DEFAULT '',
    phone             text NOT NULL DEFAULT '',
    linkedin_url      text NOT NULL DEFAULT '',
    last_contacted_on date,
    next_followup_on  date,
    notes             text NOT NULL DEFAULT '',
    created_at        timestamptz NOT NULL,
    updated_at        timestamptz NOT NULL
);

CREATE INDEX contacts_user_id_idx ON contacts (user_id);
CREATE INDEX contacts_user_updated_idx ON contacts (user_id, updated_at DESC);
CREATE INDEX contacts_user_followup_idx ON contacts (user_id, next_followup_on);
