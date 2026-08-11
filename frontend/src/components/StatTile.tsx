interface StatTileProps {
  label: string
  value: number | string
  sub: string
  alert?: boolean
}

export default function StatTile({ label, value, sub, alert }: StatTileProps) {
  return (
    <div className="card stat-tile">
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
      <div className={'stat-sub' + (alert ? ' alert' : '')}>{sub}</div>
    </div>
  )
}
