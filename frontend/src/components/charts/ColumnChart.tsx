import { useState } from 'react'
import { formatDate } from '../../lib/format'

interface Point {
  date: string // YYYY-MM-DD
  minutes: number
}

interface ColumnChartProps {
  data: Point[]
}

// Layout constants (SVG user units). Columns follow the mark spec: thin,
// 4px rounded data-end, square at the baseline, growing from one baseline.
const SLOT_W = 22
const PAD_LEFT = 40
const PAD_RIGHT = 8
const PAD_TOP = 12
const PAD_BOTTOM = 24
const PLOT_H = 150
const BAR_W = 12
const RADIUS = 4

// niceCeiling picks a clean axis maximum for minute values.
function niceCeiling(max: number): number {
  if (max <= 60) return 60
  const step = max <= 240 ? 60 : 120
  return Math.ceil(max / step) * step
}

// columnPath draws a column with a rounded top and square baseline.
function columnPath(x: number, top: number, bottom: number, w: number): string {
  const r = Math.min(RADIUS, (bottom - top) / 2, w / 2)
  return [
    `M ${x} ${bottom}`,
    `L ${x} ${top + r}`,
    `Q ${x} ${top} ${x + r} ${top}`,
    `L ${x + w - r} ${top}`,
    `Q ${x + w} ${top} ${x + w} ${top + r}`,
    `L ${x + w} ${bottom}`,
    'Z',
  ].join(' ')
}

// ColumnChart renders daily study minutes as a single-series column chart
// with a per-column hover tooltip. One series, so the card title carries the
// identity and no legend box is drawn.
export default function ColumnChart({ data }: ColumnChartProps) {
  const [hover, setHover] = useState<number | null>(null)

  const width = PAD_LEFT + data.length * SLOT_W + PAD_RIGHT
  const height = PAD_TOP + PLOT_H + PAD_BOTTOM
  const baseline = PAD_TOP + PLOT_H
  const maxValue = niceCeiling(Math.max(...data.map((d) => d.minutes), 0))
  const ticks = [0, maxValue / 2, maxValue]

  const y = (v: number) => baseline - (v / maxValue) * PLOT_H

  return (
    <div className="chart-wrap">
      <svg
        viewBox={`0 0 ${width} ${height}`}
        className="chart-svg"
        role="img"
        aria-label="Study minutes per day over the last four weeks"
      >
        {/* recessive hairline gridlines with clean tick values */}
        {ticks.map((t) => (
          <g key={t}>
            <line
              x1={PAD_LEFT}
              x2={width - PAD_RIGHT}
              y1={y(t)}
              y2={y(t)}
              className="chart-grid"
            />
            <text x={PAD_LEFT - 6} y={y(t) + 3.5} className="chart-tick" textAnchor="end">
              {t}
            </text>
          </g>
        ))}

        {/* columns */}
        {data.map((d, i) => {
          const slotX = PAD_LEFT + i * SLOT_W
          const barX = slotX + (SLOT_W - BAR_W) / 2
          return (
            <g key={d.date}>
              {d.minutes > 0 && (
                <path
                  d={columnPath(barX, y(d.minutes), baseline, BAR_W)}
                  className={'chart-col' + (hover === i ? ' hovered' : '')}
                />
              )}
              {/* hit target: the full slot, taller than the mark */}
              <rect
                x={slotX}
                y={PAD_TOP}
                width={SLOT_W}
                height={PLOT_H}
                fill="transparent"
                onMouseEnter={() => setHover(i)}
                onMouseLeave={() => setHover(null)}
              />
            </g>
          )
        })}

        {/* baseline and weekly x labels */}
        <line x1={PAD_LEFT} x2={width - PAD_RIGHT} y1={baseline} y2={baseline} className="chart-axis" />
        {data.map((d, i) =>
          i % 7 === 0 ? (
            <text
              key={d.date}
              x={PAD_LEFT + i * SLOT_W + SLOT_W / 2}
              y={baseline + 16}
              className="chart-tick"
              textAnchor="middle"
            >
              {d.date.slice(5).replace('-', '/')}
            </text>
          ) : null,
        )}
      </svg>

      {hover !== null && data[hover] && (
        <div
          className="chart-tooltip"
          style={{ left: `${((PAD_LEFT + hover * SLOT_W + SLOT_W / 2) / width) * 100}%` }}
        >
          <strong>{formatDate(data[hover].date)}</strong>
          {data[hover].minutes} min
        </div>
      )}
    </div>
  )
}
