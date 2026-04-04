import styles from './CreditGauge.module.css'

interface CreditGaugeProps {
  score?: number
  grade?: string
  ltv?: number
}

function gaugeColor(score: number): string {
  if (score >= 80) return 'var(--color-success)'
  if (score >= 60) return 'var(--color-primary)'
  if (score >= 40) return 'var(--color-warning)'
  return 'var(--color-danger)'
}

export function CreditGauge({ score, grade, ltv }: CreditGaugeProps) {
  const hasData = score !== undefined && score !== null

  const radius = 70
  const strokeWidth = 12
  const cx = 90
  const cy = 90
  const circumference = Math.PI * radius
  const fillPercent = hasData ? score / 100 : 0
  const dashOffset = circumference * (1 - fillPercent)

  return (
    <div className={styles.container}>
      <svg viewBox="0 0 180 110" className={styles.svg}>
        <path
          d={`M ${cx - radius} ${cy} A ${radius} ${radius} 0 0 1 ${cx + radius} ${cy}`}
          fill="none"
          stroke="var(--color-border)"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />
        {hasData && (
          <path
            d={`M ${cx - radius} ${cy} A ${radius} ${radius} 0 0 1 ${cx + radius} ${cy}`}
            fill="none"
            stroke={gaugeColor(score)}
            strokeWidth={strokeWidth}
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={dashOffset}
            className={styles.fill}
          />
        )}
        <text x={cx} y={cy - 14} className={styles.grade} textAnchor="middle">
          {grade ?? '--'}
        </text>
        <text x={cx} y={cy + 6} className={styles.score} textAnchor="middle">
          {hasData ? score : '--'}
        </text>
      </svg>
      {ltv !== undefined && (
        <div className={styles.ltv}>Recommended LTV: {(ltv * 100).toFixed(0)}%</div>
      )}
    </div>
  )
}
