import { useState } from 'react'
import { useReport, useReportCatalog } from '../api/insights'
import PageHeader from '../components/PageHeader'
import { formatDateTime } from '../lib/format'

// Reports lists the available reports and renders the selected one as a
// titled, timestamped table with CSV export and print.
export default function Reports() {
  const { data: catalog = [], isLoading } = useReportCatalog()
  const [selected, setSelected] = useState<string | null>(null)
  const { data: report, isFetching } = useReport(selected)

  return (
    <>
      <PageHeader title="Reports" subtitle="Generated live from your current data." />

      <div className="report-picker">
        {isLoading ? (
          <div className="loading">Loading…</div>
        ) : (
          catalog.map((info) => (
            <button
              key={info.id}
              className={'card report-card' + (selected === info.id ? ' selected' : '')}
              onClick={() => setSelected(info.id)}
            >
              <strong>{info.title}</strong>
              <span>{info.description}</span>
            </button>
          ))
        )}
      </div>

      {selected && (
        <section className="card card-pad report-view">
          {isFetching || !report ? (
            <div className="loading">Generating report…</div>
          ) : (
            <>
              <div className="report-head">
                <div>
                  <h2>{report.title}</h2>
                  <div className="rec-detail">Generated {formatDateTime(report.generatedAt)}</div>
                </div>
                <div className="report-actions">
                  <a className="btn" href={`/api/v1/reports/${report.id}?format=csv`} download>
                    Download CSV
                  </a>
                  <button className="btn" onClick={() => window.print()}>
                    Print
                  </button>
                </div>
              </div>
              {(report.rows ?? []).length === 0 ? (
                <div className="empty-state">
                  <strong>No data yet</strong>
                  Add records to populate this report.
                </div>
              ) : (
                <div className="table-wrap">
                  <table className="table report-table">
                    <thead>
                      <tr>
                        {report.columns.map((c) => (
                          <th key={c.key}>{c.label}</th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {(report.rows ?? []).map((row, i) => (
                        <tr key={i}>
                          {row.map((cell, j) => (
                            <td key={j}>{cell}</td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </section>
      )}

      {!selected && !isLoading && (
        <div className="card empty-state">
          <strong>Pick a report</strong>
          Select one of the reports above to generate it.
        </div>
      )}
    </>
  )
}
