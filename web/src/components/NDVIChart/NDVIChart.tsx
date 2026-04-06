import styles from './NDVIChart.module.css'

interface NDVIPoint {
  season: string
  ndvi_score: number
  observed_at?: string
}

interface NDVIChartProps {
  certificates: NDVIPoint[]
}

function scoreColor(score: number): string {
  if (score >= 0.6) return 'var(--color-success)'
  if (score >= 0.4) return 'var(--color-warning)'
  return 'var(--color-danger)'
}

/** Format season for display: "2025-Q2" → "Q2 '25" */
function formatSeason(s: string): string {
  const m = s.match(/^(\d{4})-?(Q\d)$/)
  if (m) return `${m[2]} '${m[1].slice(2)}`
  return s
}

/** Sort key from season string: "2025-Q2" → 20252 */
function seasonSortKey(s: string): number {
  const m = s.match(/^(\d{4})-?Q(\d)$/)
  if (m) return Number(m[1]) * 10 + Number(m[2])
  return 0
}

/** Nice Y-axis ticks that fit the data range */
function computeYTicks(min: number, max: number): number[] {
  const padding = (max - min) * 0.15 || 0.05
  const lo = Math.max(0, Math.floor((min - padding) * 20) / 20) // round down to 0.05
  const hi = Math.min(1, Math.ceil((max + padding) * 20) / 20)  // round up to 0.05
  const step = hi - lo <= 0.2 ? 0.05 : 0.1
  const ticks: number[] = []
  for (let v = lo; v <= hi + 0.001; v += step) {
    ticks.push(Math.round(v * 100) / 100)
  }
  return ticks
}

export function NDVIChart({ certificates }: NDVIChartProps) {
  if (!certificates || certificates.length === 0) {
    return <div className={styles.empty}>No NDVI data available</div>
  }

  // Sort chronologically by observed_at or season
  const sorted = [...certificates].sort((a, b) => {
    if (a.observed_at && b.observed_at) return a.observed_at.localeCompare(b.observed_at)
    return seasonSortKey(a.season) - seasonSortKey(b.season)
  })

  // Auto-scale Y-axis to data range
  const scores = sorted.map(c => c.ndvi_score)
  const dataMin = Math.min(...scores)
  const dataMax = Math.max(...scores)
  const yTicks = computeYTicks(dataMin, dataMax)
  const yMin = yTicks[0]
  const yMax = yTicks[yTicks.length - 1]
  const yRange = yMax - yMin || 0.1

  const width = 520
  const height = 200
  const padLeft = 50
  const padRight = 20
  const padTop = 12
  const padBottom = 32
  const plotW = width - padLeft - padRight
  const plotH = height - padTop - padBottom

  const points = sorted.map((c, i) => {
    const x = padLeft + (sorted.length === 1 ? plotW / 2 : (i / (sorted.length - 1)) * plotW)
    const y = padTop + plotH - ((c.ndvi_score - yMin) / yRange) * plotH
    return { x, y, ...c }
  })

  const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ')

  // Show max ~6 X-axis labels evenly spaced
  const maxLabels = 6
  const labelStep = Math.max(1, Math.ceil(points.length / maxLabels))

  return (
    <div className={styles.container}>
      <svg viewBox={`0 0 ${width} ${height}`} className={styles.svg} preserveAspectRatio="xMidYMid meet">
        {/* Y-axis grid + labels */}
        {yTicks.map((v) => {
          const y = padTop + plotH - ((v - yMin) / yRange) * plotH
          return (
            <g key={v}>
              <line x1={padLeft} y1={y} x2={width - padRight} y2={y} className={styles.gridLine} />
              <text x={padLeft - 8} y={y + 4} className={styles.axisLabel} textAnchor="end">
                {v.toFixed(2)}
              </text>
            </g>
          )
        })}

        {/* Gradient fill under line */}
        <defs>
          <linearGradient id="ndviFill" x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stopColor="var(--color-primary)" stopOpacity="0.2" />
            <stop offset="100%" stopColor="var(--color-primary)" stopOpacity="0.02" />
          </linearGradient>
        </defs>
        {points.length > 1 && (
          <path
            d={`${linePath} L ${points[points.length - 1].x} ${padTop + plotH} L ${points[0].x} ${padTop + plotH} Z`}
            fill="url(#ndviFill)"
          />
        )}

        {/* Line */}
        <path d={linePath} className={styles.line} />

        {/* Points */}
        {points.map((p, i) => (
          <circle key={i} cx={p.x} cy={p.y} r={4} fill={scoreColor(p.ndvi_score)} className={styles.dot}>
            <title>{`${p.season}: ${p.ndvi_score.toFixed(3)}`}</title>
          </circle>
        ))}

        {/* X-axis labels — horizontal, evenly spaced */}
        {points.map((p, i) =>
          i % labelStep === 0 ? (
            <text
              key={`label-${i}`}
              x={p.x}
              y={height - 6}
              className={styles.seasonLabel}
              textAnchor="middle"
            >
              {formatSeason(p.season)}
            </text>
          ) : null
        )}
      </svg>
    </div>
  )
}
