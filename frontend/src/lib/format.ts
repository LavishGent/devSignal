// Small display helpers shared across pages.

// humanize turns an enum value like "phone_screen" into "Phone screen".
export function humanize(value: string): string {
  if (!value) return ''
  const words = value.replaceAll('_', ' ')
  return words.charAt(0).toUpperCase() + words.slice(1)
}

// formatDate renders "YYYY-MM-DD" (or null) as a readable date.
export function formatDate(value: string | null): string {
  if (!value) return '—'
  const [y, m, d] = value.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

// formatDateTime renders an RFC 3339 timestamp in the viewer's locale.
export function formatDateTime(value: string | null): string {
  if (!value) return '—'
  return new Date(value).toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  })
}

// formatMoney renders a salary number compactly ("$120k").
export function formatMoney(value: number | null): string {
  if (value === null) return '—'
  if (value >= 1000) return `$${Math.round(value / 1000)}k`
  return `$${value}`
}

// toDatetimeLocal converts RFC 3339 to the value an
// <input type="datetime-local"> expects (local time, minute precision).
export function toDatetimeLocal(value: string): string {
  if (!value) return ''
  const d = new Date(value)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// fromDatetimeLocal converts a datetime-local value back to RFC 3339 UTC.
export function fromDatetimeLocal(value: string): string {
  if (!value) return ''
  return new Date(value).toISOString()
}

export function formatMinutes(min: number): string {
  if (min < 60) return `${min} min`
  const hours = min / 60
  return `${hours % 1 === 0 ? hours : hours.toFixed(1)} h`
}

export function priorityTone(priority: number): string {
  if (priority === 1) return 'badge-danger'
  if (priority === 2) return 'badge-warn'
  return ''
}
