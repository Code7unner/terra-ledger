import { useEffect, useRef, useState } from 'react'
import { useCreditProfile } from '../../../hooks/useCreditProfile'
import { useLien } from '../../../hooks/useLien'
import { CreditGauge } from '../../../components/CreditGauge/CreditGauge'
import { NDVIChart } from '../../../components/NDVIChart/NDVIChart'
import { Badge } from '../../../components/Badge/Badge'
import { Button } from '../../../components/Button/Button'
import styles from '../WizardFlow.module.css'

interface Props {
  cadastral: string
  onRestart: () => void
  showLienButton?: boolean
  onRegisterLien?: () => void
}

export function SummaryStep({ cadastral, onRestart, showLienButton, onRegisterLien }: Props) {
  const { data, loading, fetchProfile } = useCreditProfile()
  const { liens, fetchLiens } = useLien()
  const [copied, setCopied] = useState(false)
  const fetchedForRef = useRef('')

  useEffect(() => {
    if (!cadastral || fetchedForRef.current === cadastral) return
    fetchedForRef.current = cadastral
    fetchProfile(cadastral)
    fetchLiens(cadastral)
  }, [cadastral, fetchProfile, fetchLiens])

  const handleShare = async () => {
    const url = `${window.location.origin}/parcel/${encodeURIComponent(cadastral)}`
    try {
      await navigator.clipboard.writeText(url)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Fallback: no clipboard API
    }
  }

  if (loading || !data) {
    return (
      <div className={styles.stepCenter}>
        <h2 className={styles.stepTitle}>Summary</h2>
        <div className={styles.loadingAnim}>
          <div className={styles.spinner} />
          <p className={styles.loadingMsg}>Loading profile...</p>
        </div>
      </div>
    )
  }

  const ci = data.credit_intelligence
  const parcel = data.parcel
  const certs = data.productivity?.certificates ?? []
  const activeLiens = data.encumbrances?.active_liens ?? []
  const hasActiveLien = activeLiens.length > 0 || (liens ?? []).some((l) => l.status === 'active')

  return (
    <div className={styles.stepCenter}>
      <h2 className={styles.stepTitle}>Parcel Profile</h2>

      <div className={styles.summaryGrid}>
        <CreditGauge
          score={ci?.ai_score}
          grade={ci?.collateral_grade}
          ltv={ci?.recommended_ltv}
        />

        <div className={styles.summaryDetails}>
          <div className={styles.summaryRow}>
            <span className={styles.summaryLabel}>Cadastral</span>
            <span className={styles.summaryValue}>{parcel.cadastral_number}</span>
          </div>
          <div className={styles.summaryRow}>
            <span className={styles.summaryLabel}>Area</span>
            <span className={styles.summaryValue}>{parcel.area_ha} ha</span>
          </div>
          <div className={styles.summaryRow}>
            <span className={styles.summaryLabel}>Land Class</span>
            <span className={styles.summaryValue}>{parcel.land_class}</span>
          </div>
          <div className={styles.summaryRow}>
            <span className={styles.summaryLabel}>Oblast</span>
            <span className={styles.summaryValue}>{parcel.oblast ?? '-'}</span>
          </div>
          <div className={styles.summaryRow}>
            <span className={styles.summaryLabel}>KYC</span>
            <span className={styles.summaryValue}>
              {parcel.kyc_verified ? 'Verified' : 'Pending'}
            </span>
          </div>
          <div className={styles.summaryRow}>
            <span className={styles.summaryLabel}>Lien Status</span>
            <span className={styles.summaryValue}>
              {hasActiveLien ? (
                <Badge variant="active">Active Lien</Badge>
              ) : (
                <Badge variant="clean">Clean</Badge>
              )}
            </span>
          </div>
        </div>
      </div>

      {certs.length > 0 && (
        <NDVIChart certificates={certs.map((c) => ({ season: c.season, ndvi_score: c.ndvi_score }))} />
      )}

      <div className={styles.stepActions}>
        <Button variant="secondary" onClick={handleShare}>
          {copied ? 'Link Copied!' : 'Share with your bank'}
        </Button>
        <Button variant="secondary" onClick={onRestart}>
          Start New Assessment
        </Button>
        {showLienButton && onRegisterLien && (
          <Button onClick={onRegisterLien}>Register Lien</Button>
        )}
      </div>
    </div>
  )
}
