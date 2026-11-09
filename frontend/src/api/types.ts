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

// Fixed-choice fields are plain strings; the allowed values come from the
// server (see options.ts), not from this file.

// ---- records ----

export interface Application extends BaseRecord {
  company: string
  position: string
  status: string
  appliedOn: string | null
  location: string
  workMode: string
  url: string
  salaryMin: number | null
  salaryMax: number | null
  notes: string
}

export interface Contact extends BaseRecord {
  name: string
  company: string
  role: string
  relationship: string
  email: string
  phone: string
  linkedinUrl: string
  lastContactedOn: string | null
  nextFollowupOn: string | null
  notes: string
}

export interface Interview extends BaseRecord {
  applicationId: string
  stage: string
  scheduledAt: string
  durationMinutes: number
  interviewer: string
  location: string
  outcome: string
  prepNotes: string
  feedback: string
}

export interface Skill extends BaseRecord {
  name: string
  category: string
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
  status: string
  earnedOn: string | null
  expiresOn: string | null
  credentialId: string
  credentialUrl: string
  notes: string
}

export interface Project extends BaseRecord {
  name: string
  description: string
  status: string
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
  category: string
  status: string
  priority: number
  targetDate: string | null
  completedOn: string | null
  notes: string
}

// Input payloads: everything except the server-owned BaseRecord fields.
export type RecordInput<T extends BaseRecord> = Omit<T, keyof BaseRecord>
