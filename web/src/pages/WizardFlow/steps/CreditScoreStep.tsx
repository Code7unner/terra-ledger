import { useEffect, useRef, useState } from 'react'
import { useCreditProfile } from '../../../hooks/useCreditProfile'
import { CreditGauge } from '../../../components/CreditGauge/CreditGauge'
import { Button } from '../../../components/Button/Button'
import styles from '../WizardFlow.module.css'

interface Props {
  cadastral: string
  isDemo: boolean
  onNext: () => void
}

const MAX_RETRIES = 3
const RETRY_DELAY = 2000

export function CreditScoreStep({ cadastral, onNext }: Props) {
  const { data, loading, error, fetchProfile } = useCreditProfile()
  const [retryCount, setRetryCount] = useState(0)
  const fetchedForRef = useRef('')

  useEffect(() => {
    if (!cadastral || fetchedForRef.current === cadastral) return
    fetchedForRef.current = cadastral
    setRetryCount(0)
    fetchProfile(cadastral)
  }, [cadastral, fetchProfile])

  // Retry if no credit_intelligence in response
  useEffect(() => {
    if (loading || !cadastral || !data) return
    if (data.credit_intelligence) return
    if (retryCount >= MAX_RETRIES) return

    const timer = setTimeout(() => {
      setRetryCount(r => r + 1)
      fetchedForRef.current = ''
      fetchProfile(cadastral)
    }, RETRY_DELAY)

    return () => clearTimeout(timer)
  }, [data, loading, cadastral, retryCount, fetchProfile])

  const ci = data?.credit_intelligence

  if (loading || (!ci && retryCount < MAX_RETRIES)) {
    return (
      <div className={styles.stepCenter}>
        <h2 className={styles.stepTitle}>Credit Score</h2>
        <div className={styles.loadingAnim}>
          <div className={styles.spinner} />
          <p className={styles.loadingMsg}>
            {retryCount > 0
              ? `Analyzing with Claude AI... (attempt ${retryCount + 1}/${MAX_RETRIES})`
              : 'Analyzing your credit profile with Claude AI...'}
          </p>
        </div>
      </div>
    )
  }

  if (error && !ci) {
    return (
      <div className={styles.stepCenter}>
        <h2 className={styles.stepTitle}>Credit Score</h2>
        <p className={styles.error}>Failed to compute credit score: {error}</p>
        <Button variant="secondary" onClick={() => { fetchedForRef.current = ''; setRetryCount(0); fetchProfile(cadastral) }}>
          Retry
        </Button>
        <Button onClick={onNext}>Skip</Button>
      </div>
    )
  }

  const score = ci?.ai_score ?? 0
  const grade = ci?.collateral_grade ?? '--'
  const ltv = ci?.recommended_ltv ?? 0
  const explanation = ci?.explanation ?? ''
  const riskFactors = ci?.risk_factors ?? []

  return (
    <div className={styles.stepCenter}>
      <h2 className={styles.stepTitle}>Credit Score</h2>
      <p className={styles.stepDesc}>{explanation}</p>

      <CreditGauge score={score} grade={grade} ltv={ltv} />

      {riskFactors.length > 0 && (
        <div className={styles.riskFactors}>
          <strong>Risk factors:</strong>
          <ul>
            {riskFactors.map((factor, i) => (
              <li key={i}>{factor}</li>
            ))}
          </ul>
        </div>
      )}

      <span className={styles.aiAttribution}>Scored by Claude AI</span>

      <Button onClick={onNext}>Continue</Button>
    </div>
  )
}
