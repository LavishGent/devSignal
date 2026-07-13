import type { ReactNode } from 'react'

export interface Column<T> {
  key: string
  label: string
  render: (row: T) => ReactNode
}

interface DataTableProps<T> {
  columns: Column<T>[]
  rows: T[]
  loading?: boolean
  emptyTitle: string
  emptyHint: string
  onEdit: (row: T) => void
  onDelete: (row: T) => void
}

// DataTable renders module rows with edit/delete actions. Every module page
// uses it, so tables look and behave identically across the app.
export default function DataTable<T extends { id: string }>({
  columns,
  rows,
  loading,
  emptyTitle,
  emptyHint,
  onEdit,
  onDelete,
}: DataTableProps<T>) {
  if (loading) return <div className="card loading">Loading…</div>

  if (rows.length === 0) {
    return (
      <div className="card empty-state">
        <strong>{emptyTitle}</strong>
        {emptyHint}
      </div>
    )
  }

  return (
    <div className="card table-wrap">
      <table className="table">
        <thead>
          <tr>
            {columns.map((c) => (
              <th key={c.key}>{c.label}</th>
            ))}
            <th aria-label="actions" />
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.id}>
              {columns.map((c) => (
                <td key={c.key}>{c.render(row)}</td>
              ))}
              <td>
                <div className="row-actions">
                  <button className="btn btn-ghost btn-sm" onClick={() => onEdit(row)}>
                    Edit
                  </button>
                  <button className="btn btn-ghost btn-sm" onClick={() => onDelete(row)}>
                    Delete
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
