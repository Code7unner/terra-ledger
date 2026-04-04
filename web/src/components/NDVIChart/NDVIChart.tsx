import styles from './NDVIChart.module.css'

interface NDVIPoint {
  season: string
  ndvi_score: number
}

interface NDVIChartProps {
  certificates: NDVIPoint[]
}

function scoreColor(score: number): string {
  if (score >= 0.6) return 'var(--color-success)'
  if (score >= 0.4) return 'var(--color-warning)'
  return 'var(--color-danger)'
}

export function NDVIChart({ certificates }: NDVIChartProps) {
  if (!certificates || certificates.length === 0) {
    return <div className={styles.empty}>No NDVI data available</div>
  }

  const width = 400
  const height = 160
  const padX = 40
  const padY = 20
  const plotW = width - padX * 2
  const plotH = height - padY * 2

  const points = certificates.map((c, i) => {
    const x = padX + (certificates.length === 1 ? plotW / 2 : (i / (certificates.length - 1)) * plotW)
    const y = padY + plotH - c.ndvi_score * plotH
    return { x, y, ...c }
  })

  const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ')

  return (
    <div className={styles.container}>
      <svg viewBox={`0 0 ${width} ${height}`} className={styles.svg}>
        {[0, 0.25, 0.5, 0.75, 1.0].map((v) => {
          const y = padY + plotH - v * plotH
          return (
            <g key={v}>
              <line x1={padX} y1={y} x2={width - padX} y2={y} className={styles.gridLine} />
              <text x={padX - 6} y={y + 4} className={styles.axisLabel} textAnchor="end">
                {v.toFixed(1)}
              </text>
            </g>
          )
        })}

        <path d={linePath} className={styles.line} />

        {points.map((p, i) => (
          <g key={i}>
            <circle cx={p.x} cy={p.y} r={5} fill={scoreColor(p.ndvi_score)} className={styles.dot}>
              <title>{`${p.season}: ${p.ndvi_score.toFixed(2)}`}</title>
            </circle>
            <text x={p.x} y={height - 4} className={styles.seasonLabel} textAnchor="middle">
              {p.season}
            </text>
          </g>
        ))}
      </svg>
    </div>
  )
}
