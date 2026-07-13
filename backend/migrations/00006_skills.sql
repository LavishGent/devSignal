-- Technical skills with proficiency tracking.
CREATE TABLE skills (
    id                 uuid PRIMARY KEY,
    user_id            uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name               text NOT NULL,
    category           text NOT NULL CHECK (category IN (
        'language','framework','database','cloud',
        'tooling','system_design','soft_skill','other')),
    proficiency        integer NOT NULL CHECK (proficiency BETWEEN 1 AND 5),
    target_proficiency integer NOT NULL CHECK (target_proficiency BETWEEN 1 AND 5),
    last_practiced_on  date,
    notes              text NOT NULL DEFAULT '',
    created_at         timestamptz NOT NULL,
    updated_at         timestamptz NOT NULL
);

CREATE INDEX skills_user_id_idx ON skills (user_id);
CREATE INDEX skills_user_category_idx ON skills (user_id, category);
CREATE INDEX skills_user_updated_idx ON skills (user_id, updated_at DESC);
