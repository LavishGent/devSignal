import { useState, type FormEvent } from 'react'
import { ApiError } from '../api/client'
import {
  useCreateMutation,
  useDeleteMutation,
  useListQuery,
  useUpdateMutation,
} from '../api/hooks'
import {
  APPLICATION_STATUSES,
  WORK_MODES,
  type Application,
  type RecordInput,
} from '../api/types'
import ConfirmDialog from '../components/ConfirmDialog'
import DataTable, { type Column } from '../components/DataTable'
import FormField from '../components/FormField'
import Modal from '../components/Modal'
import PageHeader from '../components/PageHeader'
import StatusBadge from '../components/StatusBadge'
import { formatDate, formatMoney, humanize } from '../lib/format'
import { useDebounce } from '../lib/useDebounce'

const MODULE = 'applications'

const COLUMNS: Column<Application>[] = [
  { key: 'company', label: 'Company', render: (a) => <strong>{a.company}</strong> },
  { key: 'position', label: 'Position', render: (a) => a.position },
  { key: 'status', label: 'Status', render: (a) => <StatusBadge value={a.status} /> },
  { key: 'appliedOn', label: 'Applied', render: (a) => formatDate(a.appliedOn) },
  { key: 'workMode', label: 'Mode', render: (a) => (a.workMode ? humanize(a.workMode) : '—') },
  {
    key: 'salary',
    label: 'Salary',
    render: (a) =>
      a.salaryMin === null && a.salaryMax === null
        ? '—'
        : `${formatMoney(a.salaryMin)} – ${formatMoney(a.salaryMax)}`,
  },
]

// Applications is the job-application pipeline: searchable list, add/edit
// modal with mirrored server validation, and confirmed deletes.
export default function Applications() {
  const [q, setQ] = useState('')
  const [status, setStatus] = useState('')
  const debouncedQ = useDebounce(q)

  const { data: rows = [], isLoading } = useListQuery<Application>(MODULE, {
    q: debouncedQ,
    status,
  })

  const [editing, setEditing] = useState<Application | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Application | null>(null)
  const deleteMutation = useDeleteMutation(MODULE)

  return (
    <>
      <PageHeader
        title="Applications"
        subtitle="Every role you are pursuing, in one pipeline."
        action={
          <button className="btn btn-primary" onClick={() => setEditing('new')}>
            Add application
          </button>
        }
      />

      <div className="toolbar">
        <input
          className="input grow"
          placeholder="Search company, position, location, notes…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          aria-label="Search applications"
        />
        <select
          className="select"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          aria-label="Filter by status"
        >
          <option value="">All statuses</option>
          {APPLICATION_STATUSES.map((s) => (
            <option key={s} value={s}>
              {humanize(s)}
            </option>
          ))}
        </select>
      </div>

      <DataTable
        columns={COLUMNS}
        rows={rows}
        loading={isLoading}
        emptyTitle="No applications yet"
        emptyHint="Add the first role you're pursuing to start your pipeline."
        onEdit={setEditing}
        onDelete={setDeleting}
      />

      {editing !== null && (
        <ApplicationForm
          record={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
      {deleting !== null && (
        <ConfirmDialog
          title="Delete application"
          message={`Delete the application to ${deleting.company} (${deleting.position})? Its interviews are removed too. This cannot be undone.`}
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
  record: Application | null
  onClose: () => void
}

function ApplicationForm({ record, onClose }: FormProps) {
  const create = useCreateMutation<Application>(MODULE)
  const update = useUpdateMutation<Application>(MODULE)

  const [company, setCompany] = useState(record?.company ?? '')
  const [position, setPosition] = useState(record?.position ?? '')
  const [status, setStatus] = useState<string>(record?.status ?? 'applied')
  const [appliedOn, setAppliedOn] = useState(record?.appliedOn ?? '')
  const [location, setLocation] = useState(record?.location ?? '')
  const [workMode, setWorkMode] = useState<string>(record?.workMode ?? '')
  const [url, setUrl] = useState(record?.url ?? '')
  const [salaryMin, setSalaryMin] = useState(record?.salaryMin?.toString() ?? '')
  const [salaryMax, setSalaryMax] = useState(record?.salaryMax?.toString() ?? '')
  const [notes, setNotes] = useState(record?.notes ?? '')

  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState('')

  // validate mirrors the server's rules for instant feedback; the server
  // remains authoritative and its field errors land in the same place.
  function validate(): boolean {
    const errs: Record<string, string> = {}
    if (!company.trim()) errs.company = 'Company is required'
    if (!position.trim()) errs.position = 'Position is required'
    if (!status) errs.status = 'Status is required'
    const min = salaryMin === '' ? null : Number(salaryMin)
    const max = salaryMax === '' ? null : Number(salaryMax)
    if (min !== null && (Number.isNaN(min) || min < 0)) errs.salaryMin = 'Must be 0 or more'
    if (max !== null && (Number.isNaN(max) || max < 0)) errs.salaryMax = 'Must be 0 or more'
    if (min !== null && max !== null && max < min) errs.salaryMax = 'Must not be less than minimum'
    if (url && !/^https?:\/\//.test(url)) errs.url = 'Must start with http:// or https://'
    setErrors(errs)
    return Object.keys(errs).length === 0
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError('')
    if (!validate()) return

    const input: RecordInput<Application> = {
      company: company.trim(),
      position: position.trim(),
      status: status as Application['status'],
      appliedOn: appliedOn || null,
      location: location.trim(),
      workMode: workMode as Application['workMode'],
      url: url.trim(),
      salaryMin: salaryMin === '' ? null : Number(salaryMin),
      salaryMax: salaryMax === '' ? null : Number(salaryMax),
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
    <Modal title={record ? 'Edit application' : 'Add application'} onClose={onClose}>
      {formError ? <div className="form-error-banner" style={{ marginBottom: 14 }}>{formError}</div> : null}
      <form onSubmit={onSubmit} noValidate>
        <div className="form-grid">
          <FormField label="Company *" error={errors.company}>
            <input className="input" value={company} onChange={(e) => setCompany(e.target.value)} aria-invalid={!!errors.company} />
          </FormField>
          <FormField label="Position *" error={errors.position}>
            <input className="input" value={position} onChange={(e) => setPosition(e.target.value)} aria-invalid={!!errors.position} />
          </FormField>
          <FormField label="Status *" error={errors.status}>
            <select className="select" value={status} onChange={(e) => setStatus(e.target.value)}>
              {APPLICATION_STATUSES.map((s) => (
                <option key={s} value={s}>
                  {humanize(s)}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Applied on" error={errors.appliedOn}>
            <input className="input" type="date" value={appliedOn ?? ''} onChange={(e) => setAppliedOn(e.target.value)} />
          </FormField>
          <FormField label="Location" error={errors.location}>
            <input className="input" value={location} onChange={(e) => setLocation(e.target.value)} />
          </FormField>
          <FormField label="Work mode" error={errors.workMode}>
            <select className="select" value={workMode} onChange={(e) => setWorkMode(e.target.value)}>
              <option value="">Not specified</option>
              {WORK_MODES.map((m) => (
                <option key={m} value={m}>
                  {humanize(m)}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Salary min" error={errors.salaryMin}>
            <input className="input" type="number" min={0} value={salaryMin} onChange={(e) => setSalaryMin(e.target.value)} aria-invalid={!!errors.salaryMin} />
          </FormField>
          <FormField label="Salary max" error={errors.salaryMax}>
            <input className="input" type="number" min={0} value={salaryMax} onChange={(e) => setSalaryMax(e.target.value)} aria-invalid={!!errors.salaryMax} />
          </FormField>
          <FormField label="Job posting URL" error={errors.url} span2>
            <input className="input" type="url" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://…" aria-invalid={!!errors.url} />
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
            {busy ? 'Saving…' : record ? 'Save changes' : 'Add application'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
