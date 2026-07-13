import { useState, type FormEvent } from 'react'
import { ApiError } from '../api/client'
import {
  useCreateMutation,
  useDeleteMutation,
  useListQuery,
  useUpdateMutation,
} from '../api/hooks'
import { SKILL_CATEGORIES, type RecordInput, type Skill } from '../api/types'
import ConfirmDialog from '../components/ConfirmDialog'
import DataTable, { type Column } from '../components/DataTable'
import FormField from '../components/FormField'
import Modal from '../components/Modal'
import PageHeader from '../components/PageHeader'
import { formatDate, humanize } from '../lib/format'
import { useDebounce } from '../lib/useDebounce'

const MODULE = 'skills'

const PROFICIENCY_LEVELS = [
  { value: 1, label: '1 — Novice' },
  { value: 2, label: '2 — Advanced beginner' },
  { value: 3, label: '3 — Competent' },
  { value: 4, label: '4 — Proficient' },
  { value: 5, label: '5 — Expert' },
]

const COLUMNS: Column<Skill>[] = [
  { key: 'name', label: 'Name', render: (s) => <strong>{s.name}</strong> },
  {
    key: 'category',
    label: 'Category',
    render: (s) => <span className="badge badge-accent">{humanize(s.category)}</span>,
  },
  { key: 'proficiency', label: 'Proficiency', render: (s) => `${s.proficiency} / 5` },
  { key: 'targetProficiency', label: 'Target', render: (s) => `${s.targetProficiency} / 5` },
  {
    key: 'gap',
    label: 'Gap',
    render: (s) => {
      const gap = s.targetProficiency - s.proficiency
      return gap <= 0 ? '—' : <span className="badge badge-warn">{`+${gap}`}</span>
    },
  },
  { key: 'lastPracticedOn', label: 'Last practiced', render: (s) => formatDate(s.lastPracticedOn) },
]

// Skills is the proficiency tracker: searchable list, add/edit modal with
// mirrored server validation, and confirmed deletes.
export default function Skills() {
  const [q, setQ] = useState('')
  const [category, setCategory] = useState('')
  const debouncedQ = useDebounce(q)

  const { data: rows = [], isLoading } = useListQuery<Skill>(MODULE, {
    q: debouncedQ,
    category,
  })

  const [editing, setEditing] = useState<Skill | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Skill | null>(null)
  const deleteMutation = useDeleteMutation(MODULE)

  return (
    <>
      <PageHeader
        title="Skills"
        subtitle="Where you are, where you want to be, and the gap between."
        action={
          <button className="btn btn-primary" onClick={() => setEditing('new')}>
            Add skill
          </button>
        }
      />

      <div className="toolbar">
        <input
          className="input grow"
          placeholder="Search skills…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          aria-label="Search skills"
        />
        <select
          className="select"
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          aria-label="Filter by category"
        >
          <option value="">All categories</option>
          {SKILL_CATEGORIES.map((c) => (
            <option key={c} value={c}>
              {humanize(c)}
            </option>
          ))}
        </select>
      </div>

      <DataTable
        columns={COLUMNS}
        rows={rows}
        loading={isLoading}
        emptyTitle="No skills yet"
        emptyHint="Add a skill to start tracking your proficiency against your targets."
        onEdit={setEditing}
        onDelete={setDeleting}
      />

      {editing !== null && (
        <SkillForm record={editing === 'new' ? null : editing} onClose={() => setEditing(null)} />
      )}
      {deleting !== null && (
        <ConfirmDialog
          title="Delete skill"
          message={`Delete the skill ${deleting.name}? Study sessions keep their history (their skill link is cleared). This cannot be undone.`}
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
  record: Skill | null
  onClose: () => void
}

function SkillForm({ record, onClose }: FormProps) {
  const create = useCreateMutation<Skill>(MODULE)
  const update = useUpdateMutation<Skill>(MODULE)

  const [name, setName] = useState(record?.name ?? '')
  const [category, setCategory] = useState<string>(record?.category ?? 'language')
  const [proficiency, setProficiency] = useState(record?.proficiency.toString() ?? '1')
  const [targetProficiency, setTargetProficiency] = useState(
    record?.targetProficiency.toString() ?? '3',
  )
  const [lastPracticedOn, setLastPracticedOn] = useState(record?.lastPracticedOn ?? '')
  const [notes, setNotes] = useState(record?.notes ?? '')

  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState('')

  // validate mirrors the server's rules for instant feedback; the server
  // remains authoritative and its field errors land in the same place.
  function validate(): boolean {
    const errs: Record<string, string> = {}
    if (!name.trim()) errs.name = 'Name is required'
    const prof = Number(proficiency)
    const target = Number(targetProficiency)
    if (Number.isNaN(prof) || prof < 1 || prof > 5) errs.proficiency = 'Must be between 1 and 5'
    if (Number.isNaN(target) || target < 1 || target > 5)
      errs.targetProficiency = 'Must be between 1 and 5'
    setErrors(errs)
    return Object.keys(errs).length === 0
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError('')
    if (!validate()) return

    const input: RecordInput<Skill> = {
      name: name.trim(),
      category: category as Skill['category'],
      proficiency: Number(proficiency),
      targetProficiency: Number(targetProficiency),
      lastPracticedOn: lastPracticedOn || null,
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
    <Modal title={record ? 'Edit skill' : 'Add skill'} onClose={onClose}>
      {formError ? <div className="form-error-banner" style={{ marginBottom: 14 }}>{formError}</div> : null}
      <form onSubmit={onSubmit} noValidate>
        <div className="form-grid">
          <FormField label="Name *" error={errors.name}>
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} aria-invalid={!!errors.name} />
          </FormField>
          <FormField label="Category *" error={errors.category}>
            <select className="select" value={category} onChange={(e) => setCategory(e.target.value)}>
              {SKILL_CATEGORIES.map((c) => (
                <option key={c} value={c}>
                  {humanize(c)}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Proficiency *" error={errors.proficiency}>
            <select className="select" value={proficiency} onChange={(e) => setProficiency(e.target.value)}>
              {PROFICIENCY_LEVELS.map((l) => (
                <option key={l.value} value={l.value}>
                  {l.label}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Target proficiency *" error={errors.targetProficiency}>
            <select className="select" value={targetProficiency} onChange={(e) => setTargetProficiency(e.target.value)}>
              {PROFICIENCY_LEVELS.map((l) => (
                <option key={l.value} value={l.value}>
                  {l.label}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Last practiced on" error={errors.lastPracticedOn}>
            <input className="input" type="date" value={lastPracticedOn ?? ''} onChange={(e) => setLastPracticedOn(e.target.value)} />
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
            {busy ? 'Saving…' : record ? 'Save changes' : 'Add skill'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
