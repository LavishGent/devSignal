import { Link } from 'react-router-dom'
import { useDashboard, useRecommendations } from '../api/insights'
import { useOptions } from '../api/options'
import { useAuth } from '../auth/AuthContext'
import ColumnChart from '../components/charts/ColumnChart'
import PageHeader from '../components/PageHeader'
import StatTile from '../components/StatTile'
import { formatMinutes } from '../lib/format'
import { priorityTone } from '../lib/format'
import { formatDateTime, humanize } from '../lib/format'

export default function Dashboard() {
  const applicationStatuses = useOptions('application_status')
  const { user } = useAuth()
  const { data: summary, isLoading } = useDashboard()
  const { data: recommendations = [] } = useRecommendations(6)

  if (isLoading || !summary) return <div className="loading">Loading your dashboard…</div>

  const anyStudy = summary.studyByDay.some((d) => d.minutes > 0)
  const pipelineMax = Math.max(...Object.values(summary.applicationsByStatus), 1)

  return (
    <>
      <PageHeader
        title={`Welcome back, ${user?.displayName ?? 'there'}`}
        subtitle="Here's where your career signal stands today."
      />

      <div className="stat-grid">
        <StatTile
          label="Applications"
          value={summary.totalApplications}
          sub={`${summary.applicationsByStatus['interviewing'] ?? 0} interviewing`}
        />
        <StatTile
          label="Study time, last 30 days"
          value={formatMinutes(summary.studyMinutesLast30)}
          sub={anyStudy ? 'Keep the streak going' : 'No sessions logged yet'}
        />
        <StatTile
          label="Open goals"
          value={summary.openGoals}
          sub={summary.overdueGoals > 0 ? `${summary.overdueGoals} overdue` : 'Nothing overdue'}
          alert={summary.overdueGoals > 0}
        />
        <StatTile
          label="Certifications earned"
          value={summary.earnedCertifications}
          sub={
            summary.expiringCertifications > 0
              ? `${summary.expiringCertifications} expiring soon`
              : `${summary.activeProjects} active projects`
          }
          alert={summary.expiringCertifications > 0}
        />
      </div>

      <div className="dash-grid">
        <section className="card card-pad dash-span2">
          <h2 className="card-title">Study activity — last 4 weeks</h2>
          {anyStudy ? (
            <ColumnChart data={summary.studyByDay} />
          ) : (
            <div className="empty-state">
              <strong>No study time logged</strong>
              <Link to="/study-sessions">Log your first session</Link>
            </div>
          )}
        </section>

        <section className="card card-pad">
          <h2 className="card-title">Recommended next steps</h2>
          {recommendations.length === 0 ? (
            <div className="empty-state">
              <strong>All clear</strong>
              Nothing needs your attention right now.
            </div>
          ) : (
            <ul className="rec-list">
              {recommendations.map((r, i) => (
                <li key={i}>
                  <span className={'badge ' + priorityTone(r.priority)}>P{r.priority}</span>
                  <div>
                    <Link to={'/' + r.module}>{r.title}</Link>
                    <div className="rec-detail">{r.detail}</div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className="card card-pad">
          <h2 className="card-title">Application pipeline</h2>
          {summary.totalApplications === 0 ? (
            <div className="empty-state">
              <strong>No applications yet</strong>
              <Link to="/applications">Add your first application</Link>
            </div>
          ) : (
            <div className="hbar-list">
              {applicationStatuses.map(({ value: status, label }) => {
                const count = summary.applicationsByStatus[status] ?? 0
                if (count === 0) return null
                return (
                  <div className="hbar-row" key={status}>
                    <span className="hbar-label">{label}</span>
                    <div className="hbar-track">
                      <div className="hbar-fill" style={{ width: `${(count / pipelineMax) * 100}%` }} />
                    </div>
                    <span className="hbar-value">{count}</span>
                  </div>
                )
              })}
            </div>
          )}
        </section>

        <section className="card card-pad">
          <h2 className="card-title">Upcoming interviews</h2>
          {summary.upcomingInterviews.length === 0 ? (
            <div className="empty-state">
              <strong>Nothing scheduled</strong>
              No interviews in the next two weeks.
            </div>
          ) : (
            <ul className="upcoming-list">
              {summary.upcomingInterviews.map((iv) => (
                <li key={iv.id}>
                  <div>
                    <strong>{iv.company || 'Interview'}</strong>
                    <div className="rec-detail">
                      {iv.position ? iv.position + ' · ' : ''}
                      {humanize(iv.stage)}
                    </div>
                  </div>
                  <span className="upcoming-when">{formatDateTime(iv.scheduledAt)}</span>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className="card card-pad">
          <h2 className="card-title">Top skill gaps</h2>
          {summary.topSkillGaps.length === 0 ? (
            <div className="empty-state">
              <strong>No gaps tracked</strong>
              <Link to="/skills">Set proficiency targets</Link>
            </div>
          ) : (
            <div className="meter-list">
              {summary.topSkillGaps.map((g) => (
                <div className="meter-row" key={g.id}>
                  <div className="meter-head">
                    <span>{g.name}</span>
                    <span className="rec-detail">
                      {g.proficiency}/5 → {g.target}/5
                    </span>
                  </div>
                  <div className="meter-track">
                    <div className="meter-target" style={{ width: `${(g.target / 5) * 100}%` }} />
                    <div className="meter-fill" style={{ width: `${(g.proficiency / 5) * 100}%` }} />
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>
      </div>
    </>
  )
}

