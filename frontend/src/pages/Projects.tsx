import { useState, type FormEvent } from 'react'
import { ApiError } from '../api/client'
import {
  useCreateMutation,
  useDeleteMutation,
  useListQuery,
  useUpdateMutation,
} from '../api/hooks'
import { PROJECT_STATUSES, type Project, type RecordInput } from '../api/types'
import ConfirmDialog from '../components/ConfirmDialog'
import DataTable, { type Column } from '../components/DataTable'
import FormField from '../components/FormField'
import Modal from '../components/Modal'
import PageHeader from '../components/PageHeader'
import StatusBadge from '../components/StatusBadge'
import { formatDate, humanize } from '../lib/format'
import { useDebounce } from '../lib/useDebounce'

const MODULE = 'projects'

const COLUMNS: Column<Project>[] = [
  { key: 'name', label: 'Name', render: (p) => <strong>{p.name}</strong> },
  { key: 'status', label: 'Status', render: (p) => <StatusBadge value={p.status} /> },
  { key: 'techStack', label: 'Tech stack', render: (p) => p.techStack || '—' },
  { key: 'startedOn', label: 'Started', render: (p) => formatDate(p.startedOn) },
  { key: 'lastActivityOn', label: 'Last activity', render: (p) => formatDate(p.lastActivityOn) },
  {
    key: 'links',
    label: 'Links',
    render: (p) =>
      !p.repoUrl && !p.demoUrl ? (
        '—'
      ) : (
        <>
          {p.repoUrl && (
            <a href={p.repoUrl} target="_blank" rel="noreferrer">
              Repo
            </a>
          )}{' '}
          {p.demoUrl && (
            <a href={p.demoUrl} target="_blank" rel="noreferrer">
              Demo
            </a>
          )}
        </>
      ),
  },
]

// Projects is the portfolio tracker: searchable list, add/edit modal with
// mirrored server validation, and confirmed deletes.
export default function Projects() {
  const [q, setQ] = useState('')
  const [status, setStatus] = useState('')
  const debouncedQ = useDebounce(q)

  const { data: rows = [], isLoading } = useListQuery<Project>(MODULE, {
    q: debouncedQ,
    status,
  })

  const [editing, setEditing] = useState<Project | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Project | null>(null)
  const deleteMutation = useDeleteMutation(MODULE)

  return (
    <>
      <PageHeader
        title="Projects"
        subtitle="Portfolio work that shows what you can build."
        action={
          <button className="btn btn-primary" onClick={() => setEditing('new')}>
            Add project
          </button>
        }
      />

      <div className="toolbar">
        <input
          className="input grow"
          placeholder="Search name, description, tech stack…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          aria-label="Search projects"
        />
        <select
          className="select"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          aria-label="Filter by status"
        >
          <option value="">All statuses</option>
          {PROJECT_STATUSES.map((s) => (
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
        emptyTitle="No projects yet"
        emptyHint="Add a portfolio project to show what you can build."
        onEdit={setEditing}
        onDelete={setDeleting}
      />

      {editing !== null && (
        <ProjectForm
          record={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
      {deleting !== null && (
        <ConfirmDialog
          title="Delete project"
          message={`Delete the project "${deleting.name}"? This cannot be undone.`}
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
  record: Project | null
  onClose: () => void
}

function ProjectForm({ record, onClose }: FormProps) {
  const create = useCreateMutation<Project>(MODULE)
  const update = useUpdateMutation<Project>(MODULE)

  const [name, setName] = useState(record?.name ?? '')
  const [status, setStatus] = useState<string>(record?.status ?? 'active')
  const [techStack, setTechStack] = useState(record?.techStack ?? '')
  const [repoUrl, setRepoUrl] = useState(record?.repoUrl ?? '')
  const [demoUrl, setDemoUrl] = useState(record?.demoUrl ?? '')
  const [startedOn, setStartedOn] = useState(record?.startedOn ?? '')
  const [lastActivityOn, setLastActivityOn] = useState(record?.lastActivityOn ?? '')
  const [description, setDescription] = useState(record?.description ?? '')
  const [notes, setNotes] = useState(record?.notes ?? '')

  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState('')

  // validate mirrors the server's rules for instant feedback; the server
  // remains authoritative and its field errors land in the same place.
  function validate(): boolean {
    const errs: Record<string, string> = {}
    if (!name.trim()) errs.name = 'Name is required'
    if (!status) errs.status = 'Status is required'
    if (repoUrl && !/^https?:\/\//.test(repoUrl)) errs.repoUrl = 'Must start with http:// or https://'
    if (demoUrl && !/^https?:\/\//.test(demoUrl)) errs.demoUrl = 'Must start with http:// or https://'
    if (startedOn && lastActivityOn && lastActivityOn < startedOn)
      errs.lastActivityOn = 'Must not be before started on'
    setErrors(errs)
    return Object.keys(errs).length === 0
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError('')
    if (!validate()) return

    const input: RecordInput<Project> = {
      name: name.trim(),
      description,
      status: status as Project['status'],
      techStack: techStack.trim(),
      repoUrl: repoUrl.trim(),
      demoUrl: demoUrl.trim(),
      startedOn: startedOn || null,
      lastActivityOn: lastActivityOn || null,
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
    <Modal title={record ? 'Edit project' : 'Add project'} onClose={onClose}>
      {formError ? <div className="form-error-banner" style={{ marginBottom: 14 }}>{formError}</div> : null}
      <form onSubmit={onSubmit} noValidate>
        <div className="form-grid">
          <FormField label="Name *" error={errors.name}>
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} aria-invalid={!!errors.name} />
          </FormField>
          <FormField label="Status *" error={errors.status}>
            <select className="select" value={status} onChange={(e) => setStatus(e.target.value)}>
              {PROJECT_STATUSES.map((s) => (
                <option key={s} value={s}>
                  {humanize(s)}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Tech stack" error={errors.techStack} hint="Comma-separated, e.g. Go, React, PostgreSQL">
            <input className="input" value={techStack} onChange={(e) => setTechStack(e.target.value)} />
          </FormField>
          <FormField label="Repo URL" error={errors.repoUrl}>
            <input className="input" type="url" value={repoUrl} onChange={(e) => setRepoUrl(e.target.value)} placeholder="https://…" aria-invalid={!!errors.repoUrl} />
          </FormField>
          <FormField label="Demo URL" error={errors.demoUrl}>
            <input className="input" type="url" value={demoUrl} onChange={(e) => setDemoUrl(e.target.value)} placeholder="https://…" aria-invalid={!!errors.demoUrl} />
          </FormField>
          <FormField label="Started on" error={errors.startedOn}>
            <input className="input" type="date" value={startedOn ?? ''} onChange={(e) => setStartedOn(e.target.value)} />
          </FormField>
          <FormField label="Last activity on" error={errors.lastActivityOn}>
            <input className="input" type="date" value={lastActivityOn ?? ''} onChange={(e) => setLastActivityOn(e.target.value)} aria-invalid={!!errors.lastActivityOn} />
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
            {busy ? 'Saving…' : record ? 'Save changes' : 'Add project'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
