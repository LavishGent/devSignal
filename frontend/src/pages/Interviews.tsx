import { useState, type FormEvent } from 'react'
import { ApiError } from '../api/client'
import {
  useCreateMutation,
  useDeleteMutation,
  useListQuery,
  useUpdateMutation,
} from '../api/hooks'
import {
  INTERVIEW_OUTCOMES,
  INTERVIEW_STAGES,
  type Application,
  type Interview,
  type RecordInput,
} from '../api/types'
import ConfirmDialog from '../components/ConfirmDialog'
import DataTable, { type Column } from '../components/DataTable'
import FormField from '../components/FormField'
import Modal from '../components/Modal'
import PageHeader from '../components/PageHeader'
import StatusBadge from '../components/StatusBadge'
import { formatDateTime, fromDatetimeLocal, humanize, toDatetimeLocal } from '../lib/format'
import { useDebounce } from '../lib/useDebounce'

const MODULE = 'interviews'

// Columns are built per render because the Application cell needs the
// id → application lookup map assembled by the page.
function buildColumns(apps: Map<string, Application>): Column<Interview>[] {
  return [
    {
      key: 'application',
      label: 'Application',
      render: (i) => {
        const app = apps.get(i.applicationId)
        return app ? (
          <strong>
            {app.company} — {app.position}
          </strong>
        ) : (
          '—'
        )
      },
    },
    {
      key: 'stage',
      label: 'Stage',
      render: (i) => <span className="badge badge-accent">{humanize(i.stage)}</span>,
    },
    { key: 'scheduledAt', label: 'Scheduled', render: (i) => formatDateTime(i.scheduledAt) },
    {
      key: 'durationMinutes',
      label: 'Duration',
      render: (i) => (i.durationMinutes ? `${i.durationMinutes} min` : '—'),
    },
    { key: 'interviewer', label: 'Interviewer', render: (i) => i.interviewer || '—' },
    { key: 'outcome', label: 'Outcome', render: (i) => <StatusBadge value={i.outcome} /> },
  ]
}

// Interviews tracks every scheduled round: searchable list, add/edit
// modal with mirrored server validation, and confirmed deletes.
export default function Interviews() {
  const [q, setQ] = useState('')
  const [stage, setStage] = useState('')
  const [outcome, setOutcome] = useState('')
  const debouncedQ = useDebounce(q)

  const { data: rows = [], isLoading } = useListQuery<Interview>(MODULE, {
    q: debouncedQ,
    stage,
    outcome,
  })
  const { data: applications = [] } = useListQuery<Application>('applications', {})
  const appById = new Map(applications.map((a) => [a.id, a]))

  const [editing, setEditing] = useState<Interview | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Interview | null>(null)
  const deleteMutation = useDeleteMutation(MODULE)

  return (
    <>
      <PageHeader
        title="Interviews"
        subtitle="Every scheduled round, with prep notes and outcomes."
        action={
          <button className="btn btn-primary" onClick={() => setEditing('new')}>
            Add interview
          </button>
        }
      />

      <div className="toolbar">
        <input
          className="input grow"
          placeholder="Search interviewer, location, notes…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          aria-label="Search interviews"
        />
        <select
          className="select"
          value={stage}
          onChange={(e) => setStage(e.target.value)}
          aria-label="Filter by stage"
        >
          <option value="">All stages</option>
          {INTERVIEW_STAGES.map((s) => (
            <option key={s} value={s}>
              {humanize(s)}
            </option>
          ))}
        </select>
        <select
          className="select"
          value={outcome}
          onChange={(e) => setOutcome(e.target.value)}
          aria-label="Filter by outcome"
        >
          <option value="">All outcomes</option>
          {INTERVIEW_OUTCOMES.map((o) => (
            <option key={o} value={o}>
              {humanize(o)}
            </option>
          ))}
        </select>
      </div>

      <DataTable
        columns={buildColumns(appById)}
        rows={rows}
        loading={isLoading}
        emptyTitle="No interviews yet"
        emptyHint="Schedule your first interview to start tracking prep and outcomes."
        onEdit={setEditing}
        onDelete={setDeleting}
      />

      {editing !== null && (
        <InterviewForm
          record={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
      {deleting !== null && (
        <ConfirmDialog
          title="Delete interview"
          message={`Delete the ${humanize(deleting.stage)} interview scheduled for ${formatDateTime(deleting.scheduledAt)}? This cannot be undone.`}
          confirmLabel="Delete"
          busy={deleteMutation.isPending}
          onConfirm={() =>
            deleteMutation.mutate(deleting.id, { onSuccess: () => setDeleting(null) })
          }
          onCancel={() => setDeleting(null)}
        />
      )}
    </>
  )
}

// ---- create/edit form ----

interface FormProps {
  record: Interview | null
  onClose: () => void
}

function InterviewForm({ record, onClose }: FormProps) {
  const create = useCreateMutation<Interview>(MODULE)
  const update = useUpdateMutation<Interview>(MODULE)
  const { data: applications = [] } = useListQuery<Application>('applications', {})

  const [applicationId, setApplicationId] = useState(record?.applicationId ?? '')
  const [stage, setStage] = useState<string>(record?.stage ?? 'phone_screen')
  const [scheduledAt, setScheduledAt] = useState(toDatetimeLocal(record?.scheduledAt ?? ''))
  const [durationMinutes, setDurationMinutes] = useState(
    record?.durationMinutes ? record.durationMinutes.toString() : '',
  )
  const [interviewer, setInterviewer] = useState(record?.interviewer ?? '')
  const [location, setLocation] = useState(record?.location ?? '')
  const [outcome, setOutcome] = useState<string>(record?.outcome ?? 'pending')
  const [prepNotes, setPrepNotes] = useState(record?.prepNotes ?? '')
  const [feedback, setFeedback] = useState(record?.feedback ?? '')

  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState('')

  // validate mirrors the server's rules for instant feedback; the server
  // remains authoritative and its field errors land in the same place.
  function validate(): boolean {
    const errs: Record<string, string> = {}
    if (!applicationId) errs.applicationId = 'Application is required'
    if (!scheduledAt) errs.scheduledAt = 'Scheduled time is required'
    const duration = durationMinutes === '' ? 0 : Number(durationMinutes)
    if (Number.isNaN(duration) || duration < 0 || duration > 1440)
      errs.durationMinutes = 'Must be between 0 and 1440'
    setErrors(errs)
    return Object.keys(errs).length === 0
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError('')
    if (!validate()) return

    const input: RecordInput<Interview> = {
      applicationId,
      stage: stage as Interview['stage'],
      scheduledAt: fromDatetimeLocal(scheduledAt),
      durationMinutes: durationMinutes === '' ? 0 : Number(durationMinutes),
      interviewer: interviewer.trim(),
      location: location.trim(),
      outcome: outcome as Interview['outcome'],
      prepNotes,
      feedback,
    }

    const opts = {
      onSuccess: onClose,
      onError: (err: Error) => {
        if (err instanceof ApiError) {
          setErrors(err.fields)
          if (Object.keys(err.fields).length === 0) setFormError(err.message)
        } else {
          setFormError('Something went wrong; try again.')
        }
      },
    }
    if (record) update.mutate({ id: record.id, input }, opts)
    else create.mutate(input, opts)
  }

  const busy = create.isPending || update.isPending

  return (
    <Modal title={record ? 'Edit interview' : 'Add interview'} onClose={onClose}>
      {formError ? <div className="form-error-banner" style={{ marginBottom: 14 }}>{formError}</div> : null}
      <form onSubmit={onSubmit} noValidate>
        <div className="form-grid">
          <FormField label="Application *" error={errors.applicationId}>
            <select className="select" value={applicationId} onChange={(e) => setApplicationId(e.target.value)} aria-invalid={!!errors.applicationId}>
              <option value="">Select an application…</option>
              {applications.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.company} — {a.position}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Stage *" error={errors.stage}>
            <select className="select" value={stage} onChange={(e) => setStage(e.target.value)}>
              {INTERVIEW_STAGES.map((s) => (
                <option key={s} value={s}>
                  {humanize(s)}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Scheduled at *" error={errors.scheduledAt}>
            <input className="input" type="datetime-local" value={scheduledAt} onChange={(e) => setScheduledAt(e.target.value)} aria-invalid={!!errors.scheduledAt} />
          </FormField>
          <FormField label="Duration minutes" error={errors.durationMinutes}>
            <input className="input" type="number" min={0} max={1440} value={durationMinutes} onChange={(e) => setDurationMinutes(e.target.value)} aria-invalid={!!errors.durationMinutes} />
          </FormField>
          <FormField label="Interviewer" error={errors.interviewer}>
            <input className="input" value={interviewer} onChange={(e) => setInterviewer(e.target.value)} />
          </FormField>
          <FormField label="Location" error={errors.location}>
            <input className="input" value={location} onChange={(e) => setLocation(e.target.value)} />
          </FormField>
          <FormField label="Outcome *" error={errors.outcome}>
            <select className="select" value={outcome} onChange={(e) => setOutcome(e.target.value)}>
              {INTERVIEW_OUTCOMES.map((o) => (
                <option key={o} value={o}>
                  {humanize(o)}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Prep notes" error={errors.prepNotes} span2>
            <textarea className="textarea" value={prepNotes} onChange={(e) => setPrepNotes(e.target.value)} />
          </FormField>
          <FormField label="Feedback" error={errors.feedback} span2>
            <textarea className="textarea" value={feedback} onChange={(e) => setFeedback(e.target.value)} />
          </FormField>
        </div>
        <div className="modal-actions">
          <button type="button" className="btn" onClick={onClose} disabled={busy}>
            Cancel
          </button>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? 'Saving…' : record ? 'Save changes' : 'Add interview'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
