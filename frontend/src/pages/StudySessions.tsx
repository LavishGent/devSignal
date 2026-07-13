import { useState, type FormEvent } from 'react'
import { ApiError } from '../api/client'
import {
  useCreateMutation,
  useDeleteMutation,
  useListQuery,
  useUpdateMutation,
} from '../api/hooks'
import { type RecordInput, type Skill, type StudySession } from '../api/types'
import ConfirmDialog from '../components/ConfirmDialog'
import DataTable, { type Column } from '../components/DataTable'
import FormField from '../components/FormField'
import Modal from '../components/Modal'
import PageHeader from '../components/PageHeader'
import { formatDate } from '../lib/format'
import { useDebounce } from '../lib/useDebounce'

const MODULE = 'study-sessions'

// Skill names live in a separate module, so the columns close over a
// lookup built from the skills list at render time.
function buildColumns(skillById: Map<string, Skill>): Column<StudySession>[] {
  return [
    { key: 'topic', label: 'Topic', render: (s) => <strong>{s.topic}</strong> },
    {
      key: 'skillId',
      label: 'Skill',
      render: (s) => (s.skillId ? (skillById.get(s.skillId)?.name ?? '—') : '—'),
    },
    { key: 'minutes', label: 'Minutes', render: (s) => `${s.minutes} min` },
    { key: 'studiedOn', label: 'Studied on', render: (s) => formatDate(s.studiedOn) },
    { key: 'resource', label: 'Resource', render: (s) => s.resource || '—' },
  ]
}

// StudySessions is the study log: searchable list, add/edit modal with
// mirrored server validation, and confirmed deletes.
export default function StudySessions() {
  const [q, setQ] = useState('')
  const debouncedQ = useDebounce(q)

  const { data: rows = [], isLoading } = useListQuery<StudySession>(MODULE, {
    q: debouncedQ,
  })
  const { data: skills = [] } = useListQuery<Skill>('skills', {})
  const skillById = new Map(skills.map((s) => [s.id, s]))

  const [editing, setEditing] = useState<StudySession | 'new' | null>(null)
  const [deleting, setDeleting] = useState<StudySession | null>(null)
  const deleteMutation = useDeleteMutation(MODULE)

  return (
    <>
      <PageHeader
        title="Study sessions"
        subtitle="Every hour you invest in a skill, on the record."
        action={
          <button className="btn btn-primary" onClick={() => setEditing('new')}>
            Add session
          </button>
        }
      />

      <div className="toolbar">
        <input
          className="input grow"
          placeholder="Search topic, resource, notes…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          aria-label="Search study sessions"
        />
      </div>

      <DataTable
        columns={buildColumns(skillById)}
        rows={rows}
        loading={isLoading}
        emptyTitle="No study sessions yet"
        emptyHint="Log your first session to start tracking study time."
        onEdit={setEditing}
        onDelete={setDeleting}
      />

      {editing !== null && (
        <StudySessionForm
          record={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
      {deleting !== null && (
        <ConfirmDialog
          title="Delete study session"
          message={`Delete the "${deleting.topic}" session from ${formatDate(deleting.studiedOn)}? This cannot be undone.`}
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
  record: StudySession | null
  onClose: () => void
}

function StudySessionForm({ record, onClose }: FormProps) {
  const create = useCreateMutation<StudySession>(MODULE)
  const update = useUpdateMutation<StudySession>(MODULE)
  const { data: skills = [] } = useListQuery<Skill>('skills', {})

  const [topic, setTopic] = useState(record?.topic ?? '')
  const [skillId, setSkillId] = useState(record?.skillId ?? '')
  const [minutes, setMinutes] = useState(record?.minutes?.toString() ?? '')
  const [studiedOn, setStudiedOn] = useState(record?.studiedOn ?? '')
  const [resource, setResource] = useState(record?.resource ?? '')
  const [notes, setNotes] = useState(record?.notes ?? '')

  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState('')

  // validate mirrors the server's rules for instant feedback; the server
  // remains authoritative and its field errors land in the same place.
  function validate(): boolean {
    const errs: Record<string, string> = {}
    if (!topic.trim()) errs.topic = 'Topic is required'
    const mins = minutes === '' ? null : Number(minutes)
    if (mins === null || Number.isNaN(mins) || mins < 1 || mins > 1440)
      errs.minutes = 'Must be between 1 and 1440'
    if (!studiedOn) errs.studiedOn = 'Studied on is required'
    setErrors(errs)
    return Object.keys(errs).length === 0
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError('')
    if (!validate()) return

    const input: RecordInput<StudySession> = {
      skillId: skillId === '' ? null : skillId,
      topic: topic.trim(),
      minutes: Number(minutes),
      studiedOn,
      resource: resource.trim(),
      notes,
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
    <Modal title={record ? 'Edit study session' : 'Add study session'} onClose={onClose}>
      {formError ? <div className="form-error-banner" style={{ marginBottom: 14 }}>{formError}</div> : null}
      <form onSubmit={onSubmit} noValidate>
        <div className="form-grid">
          <FormField label="Topic *" error={errors.topic}>
            <input className="input" value={topic} onChange={(e) => setTopic(e.target.value)} aria-invalid={!!errors.topic} />
          </FormField>
          <FormField label="Skill" error={errors.skillId}>
            <select className="select" value={skillId} onChange={(e) => setSkillId(e.target.value)}>
              <option value="">No linked skill</option>
              {skills.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Minutes *" error={errors.minutes}>
            <input className="input" type="number" min={1} max={1440} value={minutes} onChange={(e) => setMinutes(e.target.value)} aria-invalid={!!errors.minutes} />
          </FormField>
          <FormField label="Studied on *" error={errors.studiedOn}>
            <input className="input" type="date" value={studiedOn} onChange={(e) => setStudiedOn(e.target.value)} aria-invalid={!!errors.studiedOn} />
          </FormField>
          <FormField label="Resource" error={errors.resource} span2>
            <input className="input" value={resource} onChange={(e) => setResource(e.target.value)} placeholder="Book, course, video…" />
          </FormField>
          <FormField label="Notes" error={errors.notes} span2>
            <textarea className="textarea" value={notes} onChange={(e) => setNotes(e.target.value)} />
          </FormField>
        </div>
        <div className="modal-actions">
          <button type="button" className="btn" onClick={onClose} disabled={busy}>
            Cancel
          </button>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? 'Saving…' : record ? 'Save changes' : 'Add session'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
