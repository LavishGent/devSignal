import { useState, type FormEvent } from 'react'
import { ApiError } from '../api/client'
import {
  useCreateMutation,
  useDeleteMutation,
  useListQuery,
  useUpdateMutation,
} from '../api/hooks'
import {
  type Goal,
  type RecordInput,
} from '../api/types'
import { useOptions } from '../api/options'
import ConfirmDialog from '../components/ConfirmDialog'
import DataTable, { type Column } from '../components/DataTable'
import FormField from '../components/FormField'
import Modal from '../components/Modal'
import PageHeader from '../components/PageHeader'
import StatusBadge from '../components/StatusBadge'
import { formatDate, humanize } from '../lib/format'
import { useDebounce } from '../lib/useDebounce'

const MODULE = 'goals'

// priorityBadge maps 1..3 to a colored pill; 1 is the most urgent.
function priorityBadge(priority: number) {
  if (priority === 1) return <span className="badge badge-danger">P1 — High</span>
  if (priority === 2) return <span className="badge badge-warn">P2 — Medium</span>
  return <span className="badge">P3 — Low</span>
}

// isOverdue flags open goals whose target date has passed.
function isOverdue(g: Goal): boolean {
  if (!g.targetDate || g.status === 'completed' || g.status === 'abandoned') return false
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  const today = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`
  return g.targetDate < today
}

const COLUMNS: Column<Goal>[] = [
  { key: 'title', label: 'Title', render: (g) => <strong>{g.title}</strong> },
  {
    key: 'category',
    label: 'Category',
    render: (g) => <span className="badge badge-accent">{humanize(g.category)}</span>,
  },
  { key: 'status', label: 'Status', render: (g) => <StatusBadge value={g.status} /> },
  { key: 'priority', label: 'Priority', render: (g) => priorityBadge(g.priority) },
  {
    key: 'targetDate',
    label: 'Target date',
    render: (g) => (
      <>
        {formatDate(g.targetDate)}
        {isOverdue(g) ? <> <span className="badge badge-danger">Overdue</span></> : null}
      </>
    ),
  },
  { key: 'completedOn', label: 'Completed', render: (g) => formatDate(g.completedOn) },
]

// Goals is the objective tracker: searchable list, add/edit modal with
// mirrored server validation, and confirmed deletes.
export default function Goals() {
  const goalStatuses = useOptions('goal_status')
  const goalCategories = useOptions('goal_category')
  const [q, setQ] = useState('')
  const [status, setStatus] = useState('')
  const [category, setCategory] = useState('')
  const debouncedQ = useDebounce(q)

  const { data: rows = [], isLoading } = useListQuery<Goal>(MODULE, {
    q: debouncedQ,
    status,
    category,
  })

  const [editing, setEditing] = useState<Goal | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Goal | null>(null)
  const deleteMutation = useDeleteMutation(MODULE)

  return (
    <>
      <PageHeader
        title="Goals"
        subtitle="What you're working toward, prioritized and dated."
        action={
          <button className="btn btn-primary" onClick={() => setEditing('new')}>
            Add goal
          </button>
        }
      />

      <div className="toolbar">
        <input
          className="input grow"
          placeholder="Search goals…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          aria-label="Search goals"
        />
        <select
          className="select"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          aria-label="Filter by status"
        >
          <option value="">All statuses</option>
          {goalStatuses.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
        </select>
        <select
          className="select"
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          aria-label="Filter by category"
        >
          <option value="">All categories</option>
          {goalCategories.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
        </select>
      </div>

      <DataTable
        columns={COLUMNS}
        rows={rows}
        loading={isLoading}
        emptyTitle="No goals yet"
        emptyHint="Add your first goal to define what success looks like."
        onEdit={setEditing}
        onDelete={setDeleting}
      />

      {editing !== null && (
        <GoalForm record={editing === 'new' ? null : editing} onClose={() => setEditing(null)} />
      )}
      {deleting !== null && (
        <ConfirmDialog
          title="Delete goal"
          message={`Delete the goal "${deleting.title}"? This cannot be undone.`}
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
  record: Goal | null
  onClose: () => void
}

function GoalForm({ record, onClose }: FormProps) {
  const goalCategories = useOptions('goal_category')
  const goalStatuses = useOptions('goal_status')
  const create = useCreateMutation<Goal>(MODULE)
  const update = useUpdateMutation<Goal>(MODULE)

  const [title, setTitle] = useState(record?.title ?? '')
  const [category, setCategory] = useState<string>(record?.category ?? 'job_search')
  const [status, setStatus] = useState<string>(record?.status ?? 'not_started')
  const [priority, setPriority] = useState(record?.priority?.toString() ?? '2')
  const [targetDate, setTargetDate] = useState(record?.targetDate ?? '')
  const [completedOn, setCompletedOn] = useState(record?.completedOn ?? '')
  const [description, setDescription] = useState(record?.description ?? '')
  const [notes, setNotes] = useState(record?.notes ?? '')

  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState('')

  // validate mirrors the server's rules for instant feedback; the server
  // remains authoritative and its field errors land in the same place.
  function validate(): boolean {
    const errs: Record<string, string> = {}
    if (!title.trim()) errs.title = 'Title is required'
    const p = Number(priority)
    if (Number.isNaN(p) || p < 1 || p > 3) errs.priority = 'Must be between 1 and 3'
    if (completedOn && status !== 'completed')
      errs.completedOn = 'May only be set when status is Completed'
    setErrors(errs)
    return Object.keys(errs).length === 0
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError('')
    if (!validate()) return

    const input: RecordInput<Goal> = {
      title: title.trim(),
      description,
      category: category as Goal['category'],
      status: status as Goal['status'],
      priority: Number(priority),
      targetDate: targetDate || null,
      completedOn: completedOn || null,
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
    <Modal title={record ? 'Edit goal' : 'Add goal'} onClose={onClose}>
      {formError ? <div className="form-error-banner" style={{ marginBottom: 14 }}>{formError}</div> : null}
      <form onSubmit={onSubmit} noValidate>
        <div className="form-grid">
          <FormField label="Title *" error={errors.title}>
            <input className="input" value={title} onChange={(e) => setTitle(e.target.value)} aria-invalid={!!errors.title} />
          </FormField>
          <FormField label="Category *" error={errors.category}>
            <select className="select" value={category} onChange={(e) => setCategory(e.target.value)}>
              {goalCategories.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Status *" error={errors.status}>
            <select className="select" value={status} onChange={(e) => setStatus(e.target.value)}>
              {goalStatuses.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Priority *" error={errors.priority}>
            <select className="select" value={priority} onChange={(e) => setPriority(e.target.value)}>
              <option value="1">1 — High</option>
              <option value="2">2 — Medium</option>
              <option value="3">3 — Low</option>
            </select>
          </FormField>
          <FormField label="Target date" error={errors.targetDate}>
            <input className="input" type="date" value={targetDate ?? ''} onChange={(e) => setTargetDate(e.target.value)} />
          </FormField>
          <FormField label="Completed on" error={errors.completedOn}>
            <input className="input" type="date" value={completedOn ?? ''} onChange={(e) => setCompletedOn(e.target.value)} aria-invalid={!!errors.completedOn} />
          </FormField>
          <FormField label="Description" error={errors.description} span2>
            <textarea className="textarea" value={description} onChange={(e) => setDescription(e.target.value)} />
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
            {busy ? 'Saving…' : record ? 'Save changes' : 'Add goal'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
