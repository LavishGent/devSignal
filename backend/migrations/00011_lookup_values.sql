-- Fixed-choice options (statuses, categories, stages) move out of the code
-- and into this table. Adding an option is an INSERT; the API validates
-- against this table at request time and the UI populates its dropdowns from
-- GET /api/v1/options, so no code changes in either half.
CREATE TABLE lookup_values (
    kind       text    NOT NULL,
    value      text    NOT NULL,
    label      text    NOT NULL,
    sort_order integer NOT NULL,
    PRIMARY KEY (kind, value)
);

CREATE INDEX lookup_values_kind_order_idx ON lookup_values (kind, sort_order);

-- The CHECK constraints on the record tables duplicated these lists; they
-- are dropped so a newly inserted option is accepted end to end. The API's
-- per-request validation against lookup_values is now the single gate.
ALTER TABLE applications   DROP CONSTRAINT applications_status_check;
ALTER TABLE applications   DROP CONSTRAINT applications_work_mode_check;
ALTER TABLE contacts       DROP CONSTRAINT contacts_relationship_check;
ALTER TABLE interviews     DROP CONSTRAINT interviews_stage_check;
ALTER TABLE interviews     DROP CONSTRAINT interviews_outcome_check;
ALTER TABLE skills         DROP CONSTRAINT skills_category_check;
ALTER TABLE certifications DROP CONSTRAINT certifications_status_check;
ALTER TABLE projects       DROP CONSTRAINT projects_status_check;
ALTER TABLE goals          DROP CONSTRAINT goals_category_check;
ALTER TABLE goals          DROP CONSTRAINT goals_status_check;

-- Initial options, identical to domain.DefaultLookups().
INSERT INTO lookup_values (kind, value, label, sort_order) VALUES
    ('application_status', 'wishlist',     'Wishlist',     1),
    ('application_status', 'applied',      'Applied',      2),
    ('application_status', 'phone_screen', 'Phone screen', 3),
    ('application_status', 'interviewing', 'Interviewing', 4),
    ('application_status', 'offer',        'Offer',        5),
    ('application_status', 'accepted',     'Accepted',     6),
    ('application_status', 'rejected',     'Rejected',     7),
    ('application_status', 'withdrawn',    'Withdrawn',    8),

    ('work_mode', 'remote', 'Remote', 1),
    ('work_mode', 'hybrid', 'Hybrid', 2),
    ('work_mode', 'onsite', 'Onsite', 3),

    ('contact_relationship', 'recruiter',      'Recruiter',      1),
    ('contact_relationship', 'hiring_manager', 'Hiring manager', 2),
    ('contact_relationship', 'engineer',       'Engineer',       3),
    ('contact_relationship', 'mentor',         'Mentor',         4),
    ('contact_relationship', 'peer',           'Peer',           5),
    ('contact_relationship', 'other',          'Other',          6),

    ('interview_stage', 'phone_screen',  'Phone screen',  1),
    ('interview_stage', 'technical',     'Technical',     2),
    ('interview_stage', 'system_design', 'System design', 3),
    ('interview_stage', 'behavioral',    'Behavioral',    4),
    ('interview_stage', 'onsite',        'Onsite',        5),
    ('interview_stage', 'final',         'Final',         6),

    ('interview_outcome', 'pending',   'Pending',   1),
    ('interview_outcome', 'passed',    'Passed',    2),
    ('interview_outcome', 'failed',    'Failed',    3),
    ('interview_outcome', 'cancelled', 'Cancelled', 4),

    ('skill_category', 'language',      'Language',      1),
    ('skill_category', 'framework',     'Framework',     2),
    ('skill_category', 'database',      'Database',      3),
    ('skill_category', 'cloud',         'Cloud',         4),
    ('skill_category', 'tooling',       'Tooling',       5),
    ('skill_category', 'system_design', 'System design', 6),
    ('skill_category', 'soft_skill',    'Soft skill',    7),
    ('skill_category', 'other',         'Other',         8),

    ('certification_status', 'planned',     'Planned',     1),
    ('certification_status', 'in_progress', 'In progress', 2),
    ('certification_status', 'earned',      'Earned',      3),
    ('certification_status', 'expired',     'Expired',     4),

    ('project_status', 'idea',      'Idea',      1),
    ('project_status', 'active',    'Active',    2),
    ('project_status', 'paused',    'Paused',    3),
    ('project_status', 'completed', 'Completed', 4),
    ('project_status', 'archived',  'Archived',  5),

    ('goal_category', 'job_search',    'Job search',    1),
    ('goal_category', 'skill',         'Skill',         2),
    ('goal_category', 'certification', 'Certification', 3),
    ('goal_category', 'project',       'Project',       4),
    ('goal_category', 'networking',    'Networking',    5),
    ('goal_category', 'other',         'Other',         6),

    ('goal_status', 'not_started', 'Not started', 1),
    ('goal_status', 'in_progress', 'In progress', 2),
    ('goal_status', 'completed',   'Completed',   3),
    ('goal_status', 'abandoned',   'Abandoned',   4);
