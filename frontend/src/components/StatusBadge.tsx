import { humanize } from '../lib/format'

// Status → visual tone. Anything unlisted renders neutral.
const TONES: Record<string, string> = {
  // applications
  applied: 'badge-info',
  phone_screen: 'badge-accent',
  interviewing: 'badge-accent',
  offer: 'badge-ok',
  accepted: 'badge-ok',
  rejected: 'badge-danger',
  withdrawn: 'badge-warn',
  // interviews
  pending: 'badge-info',
  passed: 'badge-ok',
  failed: 'badge-danger',
  cancelled: 'badge-warn',
  // certifications
  in_progress: 'badge-accent',
  earned: 'badge-ok',
  expired: 'badge-danger',
  planned: 'badge-info',
  // projects
  active: 'badge-ok',
  paused: 'badge-warn',
  completed: 'badge-ok',
  archived: 'badge-warn',
  // goals
  not_started: 'badge-info',
  abandoned: 'badge-danger',
}

// StatusBadge renders any enum value as a colored pill.
export default function StatusBadge({ value }: { value: string }) {
  const tone = TONES[value] ?? ''
  return <span className={'badge ' + tone}>{humanize(value)}</span>
}
