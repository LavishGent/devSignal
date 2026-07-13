// TypeScript mirrors of the Go domain types. JSON casing matches the
// backend exactly; date-only fields arrive as "YYYY-MM-DD" or null.

export interface BaseRecord {
  id: string
  userId: string
  createdAt: string
  updatedAt: string
}

export interface User {
  id: string
  email: string
  displayName: string
  createdAt: string
  updatedAt: string
}

// ---- enums (kept in sync with backend domain package) ----

export const APPLICATION_STATUSES = [
  'wishlist', 'applied', 'phone_screen', 'interviewing',
  'offer', 'accepted', 'rejected', 'withdrawn',
] as const
export type ApplicationStatus = (typeof APPLICATION_STATUSES)[number]

export const WORK_MODES = ['remote', 'hybrid', 'onsite'] as const
export type WorkMode = (typeof WORK_MODES)[number]

export const CONTACT_RELATIONSHIPS = [
  'recruiter', 'hiring_manager', 'engineer', 'mentor', 'peer', 'other',
] as const
export type ContactRelationship = (typeof CONTACT_RELATIONSHIPS)[number]

export const INTERVIEW_STAGES = [
  'phone_screen', 'technical', 'system_design', 'behavioral', 'onsite', 'final',
] as const
export type InterviewStage = (typeof INTERVIEW_STAGES)[number]

export const INTERVIEW_OUTCOMES = ['pending', 'passed', 'failed', 'cancelled'] as const
export type InterviewOutcome = (typeof INTERVIEW_OUTCOMES)[number]

export const SKILL_CATEGORIES = [
  'language', 'framework', 'database', 'cloud', 'tooling',
  'system_design', 'soft_skill', 'other',
] as const
export type SkillCategory = (typeof SKILL_CATEGORIES)[number]

export const CERTIFICATION_STATUSES = ['planned', 'in_progress', 'earned', 'expired'] as const
export type CertificationStatus = (typeof CERTIFICATION_STATUSES)[number]

export const PROJECT_STATUSES = ['idea', 'active', 'paused', 'completed', 'archived'] as const
export type ProjectStatus = (typeof PROJECT_STATUSES)[number]

export const GOAL_CATEGORIES = [
  'job_search', 'skill', 'certification', 'project', 'networking', 'other',
] as const
export type GoalCategory = (typeof GOAL_CATEGORIES)[number]

export const GOAL_STATUSES = ['not_started', 'in_progress', 'completed', 'abandoned'] as const
export type GoalStatus = (typeof GOAL_STATUSES)[number]

// ---- records ----

export interface Application extends BaseRecord {
  company: string
  position: string
  status: ApplicationStatus
  appliedOn: string | null
  location: string
  workMode: WorkMode | ''
  url: string
  salaryMin: number | null
  salaryMax: number | null
  notes: string
}

export interface Contact extends BaseRecord {
  name: string
  company: string
  role: string
  relationship: ContactRelationship
  email: string
  phone: string
  linkedinUrl: string
  lastContactedOn: string | null
  nextFollowupOn: string | null
  notes: string
}

export interface Interview extends BaseRecord {
  applicationId: string
  stage: InterviewStage
  scheduledAt: string
  durationMinutes: number
  interviewer: string
  location: string
  outcome: InterviewOutcome
  prepNotes: string
  feedback: string
}

export interface Skill extends BaseRecord {
  name: string
  category: SkillCategory
  proficiency: number
  targetProficiency: number
  lastPracticedOn: string | null
  notes: string
}

export interface StudySession extends BaseRecord {
  skillId: string | null
  topic: string
  minutes: number
  studiedOn: string
  resource: string
  notes: string
}

export interface Certification extends BaseRecord {
  name: string
  issuer: string
  status: CertificationStatus
  earnedOn: string | null
  expiresOn: string | null
  credentialId: string
  credentialUrl: string
  notes: string
}

export interface Project extends BaseRecord {
  name: string
  description: string
  status: ProjectStatus
  techStack: string
  repoUrl: string
  demoUrl: string
  startedOn: string | null
  lastActivityOn: string | null
  notes: string
}

export interface Goal extends BaseRecord {
  title: string
  description: string
  category: GoalCategory
  status: GoalStatus
  priority: number
  targetDate: string | null
  completedOn: string | null
  notes: string
}

// Input payloads: everything except the server-owned BaseRecord fields.
export type RecordInput<T extends BaseRecord> = Omit<T, keyof BaseRecord>
