import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card } from '../../components/Card/Card'
import { Input } from '../../components/Input/Input'
import { Button } from '../../components/Button/Button'
import { MetricCard } from '../../components/MetricCard/MetricCard'
import styles from './LenderDashboard.module.css'

export function LenderDashboard() {
  const [cadastral, setCadastral] = useState('')
  const navigate = useNavigate()

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (cadastral.trim()) {
      navigate(`/parcel/${encodeURIComponent(cadastral.trim())}`)
    }
  }

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>Lender Dashboard</h1>
      <p className={styles.subtitle}>Search parcels and assess credit profiles</p>

      <Card padding="lg">
        <form className={styles.searchForm} onSubmit={handleSearch}>
          <Input
            label="Cadastral Number"
            placeholder="Enter cadastral number..."
            value={cadastral}
            onChange={(e) => setCadastral(e.target.value)}
          />
          <Button type="submit" variant="primary" disabled={!cadastral.trim()}>
            Search
          </Button>
        </form>
      </Card>

      <div className={styles.metrics}>
        <MetricCard label="Active Liens" value="--" />
        <MetricCard label="Total Parcels" value="--" />
        <MetricCard label="Clean Parcels" value="--" />
      </div>

      <Card padding="md">
        <div className={styles.placeholder}>
          Search results will appear here
        </div>
      </Card>
    </div>
  )
}
