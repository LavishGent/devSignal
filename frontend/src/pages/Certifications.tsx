import { useState, type FormEvent } from 'react'
import { ApiError } from '../api/client'
import {
  useCreateMutation,
  useDeleteMutation,
  useListQuery,
  useUpdateMutation,
} from '../api/hooks'
import {
  CERTIFICATION_STATUSES,
  type Certification,
  type RecordInput,
} from '../api/types'
import ConfirmDialog from '../components/ConfirmDialog'
import DataTable, { type Column } from '../components/DataTable'
import FormField from '../components/FormField'
import Modal from '../components/Modal'
import PageHeader from '../components/PageHeader'
import StatusBadge from '../components/StatusBadge'
import { formatDate, humanize } from '../lib/format'
import { useDebounce } from '../lib/useDebounce'

const MODULE = 'certifications'

const COLUMNS: Column<Certification>[] = [
  { key: 'name', label: 'Name', render: (c) => <strong>{c.name}</strong> },
  { key: 'issuer', label: 'Issuer', render: (c) => c.issuer },
  { key: 'status', label: 'Status', render: (c) => <StatusBadge value={c.status} /> },
  { key: 'earnedOn', label: 'Earned', render: (c) => formatDate(c.earnedOn) },
  { key: 'expiresOn', label: 'Expires', render: (c) => formatDate(c.expiresOn) },
  {
    key: 'credential',
    label: 'Credential',
    render: (c) =>
      c.credentialUrl ? (
        <a href={c.credentialUrl} target="_blank" rel="noreferrer">
          View
        </a>
      ) : (
        c.credentialId || '—'
      ),
  },
]

// Certifications tracks credentials planned and earned: searchable list,
// add/edit modal with mirrored server validation, and confirmed deletes.
export default function Certifications() {
  const [q, setQ] = useState('')
  const [status, setStatus] = useState('')
  const debouncedQ = useDebounce(q)

  const { data: rows = [], isLoading } = useListQuery<Certification>(MODULE, {
    q: debouncedQ,
    status,
  })

  const [editing, setEditing] = useState<Certification | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Certification | null>(null)
  const deleteMutation = useDeleteMutation(MODULE)

  return (
    <>
      <PageHeader
        title="Certifications"
        subtitle="Credentials you hold and the ones you're working toward."
        action={
          <button className="btn btn-primary" onClick={() => setEditing('new')}>
            Add certification
          </button>
        }
      />

      <div className="toolbar">
        <input
          className="input grow"
          placeholder="Search name, issuer, notes…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          aria-label="Search certifications"
        />
        <select
          className="select"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          aria-label="Filter by status"
        >
          <option value="">All statuses</option>
          {CERTIFICATION_STATUSES.map((s) => (
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
        emptyTitle="No certifications yet"
        emptyHint="Add a credential you hold or plan to earn."
        onEdit={setEditing}
        onDelete={setDeleting}
      />

      {editing !== null && (
        <CertificationForm
          record={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
      {deleting !== null && (
        <ConfirmDialog
          title="Delete certification"
          message={`Delete the certification ${deleting.name}? This cannot be undone.`}
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
  record: Certification | null
  onClose: () => void
}

function CertificationForm({ record, onClose }: FormProps) {
  const create = useCreateMutation<Certification>(MODULE)
  const update = useUpdateMutation<Certification>(MODULE)

  const [name, setName] = useState(record?.name ?? '')
  const [issuer, setIssuer] = useState(record?.issuer ?? '')
  const [status, setStatus] = useState<string>(record?.status ?? 'planned')
  const [earnedOn, setEarnedOn] = useState(record?.earnedOn ?? '')
  const [expiresOn, setExpiresOn] = useState(record?.expiresOn ?? '')
  const [credentialId, setCredentialId] = useState(record?.credentialId ?? '')
  const [credentialUrl, setCredentialUrl] = useState(record?.credentialUrl ?? '')
  const [notes, setNotes] = useState(record?.notes ?? '')

  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState('')

  // validate mirrors the server's rules for instant feedback; the server
  // remains authoritative and its field errors land in the same place.
  function validate(): boolean {
    const errs: Record<string, string> = {}
    if (!name.trim()) errs.name = 'Name is required'
    if (!issuer.trim()) errs.issuer = 'Issuer is required'
    if (!status) errs.status = 'Status is required'
    if (status === 'earned' && !earnedOn) errs.earnedOn = 'Required when status is earned'
    if (earnedOn && expiresOn && expiresOn < earnedOn)
      errs.expiresOn = 'Must not be before earned date'
    if (credentialUrl && !/^https?:\/\//.test(credentialUrl))
      errs.credentialUrl = 'Must start with http:// or https://'
    setErrors(errs)
    return Object.keys(errs).length === 0
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError('')
    if (!validate()) return

    const input: RecordInput<Certification> = {
      name: name.trim(),
      issuer: issuer.trim(),
      status: status as Certification['status'],
      earnedOn: earnedOn || null,
      expiresOn: expiresOn || null,
      credentialId: credentialId.trim(),
      credentialUrl: credentialUrl.trim(),
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
    <Modal title={record ? 'Edit certification' : 'Add certification'} onClose={onClose}>
      {formError ? <div className="form-error-banner" style={{ marginBottom: 14 }}>{formError}</div> : null}
      <form onSubmit={onSubmit} noValidate>
        <div className="form-grid">
          <FormField label="Name *" error={errors.name}>
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} aria-invalid={!!errors.name} />
          </FormField>
          <FormField label="Issuer *" error={errors.issuer}>
            <input className="input" value={issuer} onChange={(e) => setIssuer(e.target.value)} aria-invalid={!!errors.issuer} />
          </FormField>
          <FormField label="Status *" error={errors.status}>
            <select className="select" value={status} onChange={(e) => setStatus(e.target.value)}>
              {CERTIFICATION_STATUSES.map((s) => (
                <option key={s} value={s}>
                  {humanize(s)}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Earned on" error={errors.earnedOn}>
            <input className="input" type="date" value={earnedOn ?? ''} onChange={(e) => setEarnedOn(e.target.value)} aria-invalid={!!errors.earnedOn} />
          </FormField>
          <FormField label="Expires on" error={errors.expiresOn}>
            <input className="input" type="date" value={expiresOn ?? ''} onChange={(e) => setExpiresOn(e.target.value)} aria-invalid={!!errors.expiresOn} />
          </FormField>
          <FormField label="Credential ID" error={errors.credentialId}>
            <input className="input" value={credentialId} onChange={(e) => setCredentialId(e.target.value)} />
          </FormField>
          <FormField label="Credential URL" error={errors.credentialUrl} span2>
            <input className="input" type="url" value={credentialUrl} onChange={(e) => setCredentialUrl(e.target.value)} placeholder="https://…" aria-invalid={!!errors.credentialUrl} />
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
            {busy ? 'Saving…' : record ? 'Save changes' : 'Add certification'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
