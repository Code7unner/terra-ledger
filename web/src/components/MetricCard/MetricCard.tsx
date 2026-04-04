import { Card } from '../Card/Card'
import styles from './MetricCard.module.css'

interface MetricCardProps {
  label: string
  value: string
  trend?: string
}

export function MetricCard({ label, value, trend }: MetricCardProps) {
  return (
    <Card padding="md">
      <div className={styles.metric}>
        <span className={styles.label}>{label}</span>
        <span className={styles.value}>{value}</span>
        {trend && <span className={styles.trend}>{trend}</span>}
      </div>
    </Card>
  )
}
