import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useGlobalSearch } from '../api/insights'
import PageHeader from '../components/PageHeader'
import { formatDateTime } from '../lib/format'
import { useDebounce } from '../lib/useDebounce'

// Search queries every module at once and shows grouped, navigable results.
export default function Search() {
  const [q, setQ] = useState('')
  const debouncedQ = useDebounce(q, 300)
  const { data: groups = [], isFetching } = useGlobalSearch(debouncedQ)

  const active = debouncedQ.trim().length > 0

  return (
    <>
      <PageHeader title="Search" subtitle="Search across everything you track." />

      <div className="toolbar">
        <input
          className="input grow"
          placeholder="Search applications, contacts, skills, projects…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          autoFocus
          aria-label="Global search"
        />
      </div>

      {!active ? (
        <div className="card empty-state">
          <strong>Start typing to search</strong>
          Results from every module appear here, grouped by type.
        </div>
      ) : isFetching && groups.length === 0 ? (
        <div className="loading">Searching…</div>
      ) : groups.length === 0 ? (
        <div className="card empty-state">
          <strong>No matches for “{debouncedQ}”</strong>
          Try a different term.
        </div>
      ) : (
        groups.map((group) => (
          <section className="card search-group" key={group.module}>
            <h2 className="search-group-title">
              {group.label} <span className="badge">{group.items.length}</span>
            </h2>
            <ul className="search-items">
              {group.items.map((item) => (
                <li key={item.id}>
                  <Link to={'/' + group.module}>
                    <strong>{item.title}</strong>
                    <span className="rec-detail">{item.subtitle}</span>
                  </Link>
                  <span className="upcoming-when">{formatDateTime(item.updatedAt)}</span>
                </li>
              ))}
            </ul>
          </section>
        ))
      )}
    </>
  )
}
