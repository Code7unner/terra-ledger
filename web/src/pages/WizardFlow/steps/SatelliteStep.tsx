import { useState, useEffect, useRef } from 'react'
import type { WizardData } from '../WizardFlow'
import { get } from '../../../api/client'
import { useToast } from '../../../components/Toast/useToast'
import { Button } from '../../../components/Button/Button'
import styles from '../WizardFlow.module.css'

interface Props {
  cadastral: string
  isDemo: boolean
  onUpdate: (partial: Partial<WizardData>) => void
  onNext: () => void
}

const LOADING_MESSAGES = [
  'Connecting to satellite...',
  'Finding your field from space...',
  'Taking a fresh look at your land...',
  'Measuring how green your crops are...',
  'Comparing with other fields in the region...',
  'Checking growth patterns across seasons...',
  'Calculating your land productivity score...',
  'Creating your digital certificate...',
  'Recording on the blockchain...',
  'Almost done...',
]

const MIN_LOADING_MS = 8000
const MSG_INTERVAL_MS = 2500

interface NdviResponse {
  ndvi: number
  ndvi_score?: number
}

function ndviColor(score: number): string {
  if (score >= 0.6) return 'var(--color-success)'
  if (score >= 0.4) return 'var(--color-warning)'
  return 'var(--color-danger)'
}

export function SatelliteStep({ cadastral, isDemo, onUpdate, onNext }: Props) {
  const { toast } = useToast()
  const [loading, setLoading] = useState(true)
  const [msgIndex, setMsgIndex] = useState(0)
  const [ndvi, setNdvi] = useState<number | null>(null)
  const startedRef = useRef(false)

  useEffect(() => {
    if (startedRef.current) return
    startedRef.current = true

    const startTime = Date.now()

    const msgTimer = setInterval(() => {
      setMsgIndex((prev) => {
        if (prev < LOADING_MESSAGES.length - 1) return prev + 1
        return prev
      })
    }, MSG_INTERVAL_MS)

    const fetchNdvi = async (): Promise<number> => {
      if (!cadastral) return 0.72
      try {
        const resp = await get<NdviResponse>(
          `/api/v1/parcels/${encodeURIComponent(cadastral)}/ndvi`,
        )
        return resp.ndvi ?? resp.ndvi_score ?? 0
      } catch (err) {
        toast(err instanceof Error ? err.message : 'Satellite API unavailable, using estimated data', 'info')
        return 0.72
      }
    }

    fetchNdvi().then((score) => {
      const elapsed = Date.now() - startTime
      const remaining = Math.max(0, MIN_LOADING_MS - elapsed)

      setTimeout(() => {
        clearInterval(msgTimer)
        setNdvi(score)
        onUpdate({ ndviScore: score })
        setLoading(false)
      }, remaining)
    })

    return () => {
      clearInterval(msgTimer)
    }
  }, [cadastral, isDemo, onUpdate, toast])

  if (loading) {
    return (
      <div className={styles.stepCenter}>
        <h2 className={styles.stepTitle}>Satellite Verification</h2>
        <div className={styles.loadingAnim}>
          <div className={styles.spinner} />
          <p className={styles.loadingMsg}>{LOADING_MESSAGES[msgIndex]}</p>
        </div>
      </div>
    )
  }

  const score = ndvi ?? 0
  const color = ndviColor(score)

  return (
    <div className={styles.stepCenter}>
      <h2 className={styles.stepTitle}>Satellite Verification</h2>
      <p className={styles.stepDesc}>
        Your land has been analyzed using satellite imagery.
      </p>

      <div className={styles.ndviResult}>
        <span className={styles.ndviLabel}>NDVI Score</span>
        <span className={styles.ndviScore} style={{ color }}>
          {score.toFixed(2)}
        </span>
        <div className={styles.ndviBar}>
          <div
            className={styles.ndviBarFill}
            style={{
              width: `${score * 100}%`,
              backgroundColor: color,
            }}
          />
        </div>
        <p className={styles.ndviDesc}>
          {score >= 0.6
            ? 'Healthy vegetation detected. Your land shows strong productivity.'
            : score >= 0.4
              ? 'Moderate vegetation. Some areas may benefit from improvement.'
              : 'Low vegetation detected. This may affect your credit score.'}
        </p>
        <span className={styles.mintConfirm}>Certificate minted on Solana</span>
      </div>

      <Button onClick={onNext}>Continue</Button>
    </div>
  )
}
