import { useEffect, useRef } from 'react'
import { useCreditProfile } from '../../../hooks/useCreditProfile'
import { CreditGauge } from '../../../components/CreditGauge/CreditGauge'
import { Button } from '../../../components/Button/Button'
import styles from '../WizardFlow.module.css'

interface Props {
  cadastral: string
  isDemo: boolean
  onNext: () => void
}

const MOCK_SCORE = 76
const MOCK_GRADE = 'B+'
const MOCK_LTV = 0.65
const MOCK_EXPLANATION =
  'This parcel demonstrates good productivity based on satellite verification. ' +
  'The NDVI trend is stable with moderate land class, resulting in a solid credit profile.'
const MOCK_RISK_FACTORS = [
  'Land class 2 limits maximum LTV',
  'Single season of NDVI data available',
  'No historical lien records found',
]

export function CreditScoreStep({ cadastral, isDemo, onNext }: Props) {
  const { data, loading, error, fetchProfile } = useCreditProfile()
  const fetchedRef = useRef(false)

  useEffect(() => {
    if (fetchedRef.current || !cadastral) return
    fetchedRef.current = true
    fetchProfile(cadastral)
  }, [cadastral, fetchProfile])

  const ci = data?.credit_intelligence
  const useMock = !cadastral || (isDemo && (error || !ci))

  const score = useMock ? MOCK_SCORE : (ci?.ai_score ?? 0)
  const grade = useMock ? MOCK_GRADE : (ci?.collateral_grade ?? '--')
  const ltv = useMock ? MOCK_LTV : (ci?.recommended_ltv ?? 0)
  const explanation = useMock ? MOCK_EXPLANATION : (ci?.explanation ?? '')
  const riskFactors = useMock ? MOCK_RISK_FACTORS : (ci?.risk_factors ?? [])

  if (loading) {
    return (
      <div className={styles.stepCenter}>
        <h2 className={styles.stepTitle}>Credit Score</h2>
        <div className={styles.loadingAnim}>
          <div className={styles.spinner} />
          <p className={styles.loadingMsg}>Analyzing your credit profile...</p>
        </div>
      </div>
    )
  }

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
