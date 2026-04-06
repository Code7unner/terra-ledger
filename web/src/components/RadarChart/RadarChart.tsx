import styles from './RadarChart.module.css'

interface Props {
  ndvi: number
  ndwi: number
  evi: number
  lai: number
  size?: number
}

const LABELS = ['NDVI', 'EVI', 'NDWI', 'LAI']

export function RadarChart({ ndvi, ndwi, evi, lai, size = 200 }: Props) {
  const values = [ndvi, evi, ndwi, lai]
  const cx = size / 2
  const cy = size / 2
  const r = size * 0.38

  const angle = (i: number) => (Math.PI * 2 * i) / 4 - Math.PI / 2
  const point = (i: number, val: number) => ({
    x: cx + r * val * Math.cos(angle(i)),
    y: cy + r * val * Math.sin(angle(i)),
  })

  const gridLevels = [0.25, 0.5, 0.75, 1.0]
  const dataPoints = values.map((v, i) => point(i, v))
  const polygon = dataPoints.map(p => `${p.x},${p.y}`).join(' ')

  const refPoints = [0, 1, 2, 3].map(i => point(i, 0.7))
  const refPolygon = refPoints.map(p => `${p.x},${p.y}`).join(' ')

  return (
    <div className={styles.container}>
      <svg viewBox={`0 0 ${size} ${size}`} className={styles.svg}>
        {gridLevels.map(level => {
          const pts = [0, 1, 2, 3].map(i => point(i, level))
          return (
            <polygon
              key={level}
              points={pts.map(p => `${p.x},${p.y}`).join(' ')}
              className={styles.gridLine}
            />
          )
        })}
        {[0, 1, 2, 3].map(i => {
          const end = point(i, 1)
          return (
            <line
              key={i}
              x1={cx}
              y1={cy}
              x2={end.x}
              y2={end.y}
              className={styles.axis}
            />
          )
        })}
        <polygon points={refPolygon} className={styles.refArea} />
        <polygon points={polygon} className={styles.dataArea} />
        {dataPoints.map((p, i) => (
          <circle key={i} cx={p.x} cy={p.y} r="3" className={styles.dataPoint} />
        ))}
        {LABELS.map((label, i) => {
          const lp = point(i, 1.2)
          return (
            <text
              key={label}
              x={lp.x}
              y={lp.y}
              className={styles.label}
              textAnchor="middle"
              dominantBaseline="middle"
            >
              {label}
            </text>
          )
        })}
      </svg>
    </div>
  )
}
