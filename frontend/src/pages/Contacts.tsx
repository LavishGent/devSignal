import { useState, type FormEvent } from 'react'
import { ApiError } from '../api/client'
import {
  useCreateMutation,
  useDeleteMutation,
  useListQuery,
  useUpdateMutation,
} from '../api/hooks'
import {
  type Contact,
  type RecordInput,
} from '../api/types'
import { useOptions } from '../api/options'
import ConfirmDialog from '../components/ConfirmDialog'
import DataTable, { type Column } from '../components/DataTable'
import FormField from '../components/FormField'
import Modal from '../components/Modal'
import PageHeader from '../components/PageHeader'
import { formatDate, humanize } from '../lib/format'
import { useDebounce } from '../lib/useDebounce'

const MODULE = 'contacts'

const COLUMNS: Column<Contact>[] = [
  { key: 'name', label: 'Name', render: (c) => <strong>{c.name}</strong> },
  { key: 'company', label: 'Company', render: (c) => c.company },
  {
    key: 'relationship',
    label: 'Relationship',
    render: (c) => <span className="badge badge-accent">{humanize(c.relationship)}</span>,
  },
  {
    key: 'email',
    label: 'Email',
    render: (c) => (c.email ? <a href={`mailto:${c.email}`}>{c.email}</a> : '—'),
  },
  { key: 'lastContactedOn', label: 'Last contacted', render: (c) => formatDate(c.lastContactedOn) },
  { key: 'nextFollowupOn', label: 'Next follow-up', render: (c) => formatDate(c.nextFollowupOn) },
]

// Contacts is the networking rolodex: searchable list, add/edit modal with
// mirrored server validation, and confirmed deletes.
export default function Contacts() {
  const contactRelationships = useOptions('contact_relationship')
  const [q, setQ] = useState('')
  const [relationship, setRelationship] = useState('')
  const debouncedQ = useDebounce(q)

  const { data: rows = [], isLoading } = useListQuery<Contact>(MODULE, {
    q: debouncedQ,
    relationship,
  })

  const [editing, setEditing] = useState<Contact | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Contact | null>(null)
  const deleteMutation = useDeleteMutation(MODULE)

  return (
    <>
      <PageHeader
        title="Contacts"
        subtitle="Everyone in your network, and when to reach out next."
        action={
          <button className="btn btn-primary" onClick={() => setEditing('new')}>
            Add contact
          </button>
        }
      />

      <div className="toolbar">
        <input
          className="input grow"
          placeholder="Search name, company, role, email, notes…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          aria-label="Search contacts"
        />
        <select
          className="select"
          value={relationship}
          onChange={(e) => setRelationship(e.target.value)}
          aria-label="Filter by relationship"
        >
          <option value="">All relationships</option>
          {contactRelationships.map((o) => (
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
        emptyTitle="No contacts yet"
        emptyHint="Add the first person in your network to start tracking follow-ups."
        onEdit={setEditing}
        onDelete={setDeleting}
      />

      {editing !== null && (
        <ContactForm
          record={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
      {deleting !== null && (
        <ConfirmDialog
          title="Delete contact"
          message={`Delete ${deleting.name} from your contacts? This cannot be undone.`}
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
  record: Contact | null
  onClose: () => void
}

function ContactForm({ record, onClose }: FormProps) {
  const contactRelationships = useOptions('contact_relationship')
  const create = useCreateMutation<Contact>(MODULE)
  const update = useUpdateMutation<Contact>(MODULE)

  const [name, setName] = useState(record?.name ?? '')
  const [relationship, setRelationship] = useState<string>(record?.relationship ?? 'recruiter')
  const [company, setCompany] = useState(record?.company ?? '')
  const [role, setRole] = useState(record?.role ?? '')
  const [email, setEmail] = useState(record?.email ?? '')
  const [phone, setPhone] = useState(record?.phone ?? '')
  const [linkedinUrl, setLinkedinUrl] = useState(record?.linkedinUrl ?? '')
  const [lastContactedOn, setLastContactedOn] = useState(record?.lastContactedOn ?? '')
  const [nextFollowupOn, setNextFollowupOn] = useState(record?.nextFollowupOn ?? '')
  const [notes, setNotes] = useState(record?.notes ?? '')

  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState('')

  // validate mirrors the server's rules for instant feedback; the server
  // remains authoritative and its field errors land in the same place.
  function validate(): boolean {
    const errs: Record<string, string> = {}
    if (!name.trim()) errs.name = 'Name is required'
    if (!relationship) errs.relationship = 'Relationship is required'
    if (email && !/.+@.+/.test(email)) errs.email = 'Must be a valid email address'
    if (linkedinUrl && !/^https?:\/\//.test(linkedinUrl))
      errs.linkedinUrl = 'Must start with http:// or https://'
    setErrors(errs)
    return Object.keys(errs).length === 0
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError('')
    if (!validate()) return

    const input: RecordInput<Contact> = {
      name: name.trim(),
      company: company.trim(),
      role: role.trim(),
      relationship: relationship as Contact['relationship'],
      email: email.trim(),
      phone: phone.trim(),
      linkedinUrl: linkedinUrl.trim(),
      lastContactedOn: lastContactedOn || null,
      nextFollowupOn: nextFollowupOn || null,
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
    <Modal title={record ? 'Edit contact' : 'Add contact'} onClose={onClose}>
      {formError ? <div className="form-error-banner" style={{ marginBottom: 14 }}>{formError}</div> : null}
      <form onSubmit={onSubmit} noValidate>
        <div className="form-grid">
          <FormField label="Name *" error={errors.name}>
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} aria-invalid={!!errors.name} />
          </FormField>
          <FormField label="Relationship *" error={errors.relationship}>
            <select className="select" value={relationship} onChange={(e) => setRelationship(e.target.value)}>
              {contactRelationships.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Company" error={errors.company}>
            <input className="input" value={company} onChange={(e) => setCompany(e.target.value)} />
          </FormField>
          <FormField label="Role" error={errors.role}>
            <input className="input" value={role} onChange={(e) => setRole(e.target.value)} />
          </FormField>
          <FormField label="Email" error={errors.email}>
            <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} aria-invalid={!!errors.email} />
          </FormField>
          <FormField label="Phone" error={errors.phone}>
            <input className="input" type="tel" value={phone} onChange={(e) => setPhone(e.target.value)} />
          </FormField>
          <FormField label="LinkedIn URL" error={errors.linkedinUrl} span2>
            <input className="input" type="url" value={linkedinUrl} onChange={(e) => setLinkedinUrl(e.target.value)} placeholder="https://…" aria-invalid={!!errors.linkedinUrl} />
          </FormField>
          <FormField label="Last contacted on" error={errors.lastContactedOn}>
            <input className="input" type="date" value={lastContactedOn ?? ''} onChange={(e) => setLastContactedOn(e.target.value)} />
          </FormField>
          <FormField label="Next follow-up on" error={errors.nextFollowupOn}>
            <input className="input" type="date" value={nextFollowupOn ?? ''} onChange={(e) => setNextFollowupOn(e.target.value)} />
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
            {busy ? 'Saving…' : record ? 'Save changes' : 'Add contact'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
