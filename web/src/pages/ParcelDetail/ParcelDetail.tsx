import { useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Card } from '../../components/Card/Card'
import { Badge } from '../../components/Badge/Badge'
import { MetricCard } from '../../components/MetricCard/MetricCard'
import { Skeleton } from '../../components/Skeleton/Skeleton'
import { NDVIChart } from '../../components/NDVIChart/NDVIChart'
import { CreditGauge } from '../../components/CreditGauge/CreditGauge'
import { useCreditProfile } from '../../hooks/useCreditProfile'
import styles from './ParcelDetail.module.css'

export default function ParcelDetail() {
  const { cadastral } = useParams<{ cadastral: string }>()
  const { data, loading, error, fetchProfile } = useCreditProfile()

  useEffect(() => {
    if (cadastral) fetchProfile(cadastral)
  }, [cadastral, fetchProfile])

  if (loading) {
    return (
      <div className={styles.page}>
        <Skeleton width="300px" height="36px" />
        <div className={styles.metrics}>
          <Skeleton height="80px" />
          <Skeleton height="80px" />
          <Skeleton height="80px" />
        </div>
        <Skeleton height="200px" />
      </div>
    )
  }

  if (error) {
    return (
      <div className={styles.page}>
        <div className={styles.errorState} role="alert">
          <h2>Failed to load parcel</h2>
          <p>{error}</p>
        </div>
      </div>
    )
  }

  if (!data) {
    return (
      <div className={styles.page}>
        <h1 className={styles.title}>Parcel {cadastral}</h1>
        <div className={styles.placeholder}>
          <p className={styles.placeholderTitle}>No data available</p>
          <p className={styles.placeholderHint}>
            This parcel may not be registered yet or the backend is unreachable
          </p>
        </div>
      </div>
    )
  }

  const { parcel, productivity, encumbrances, credit_intelligence } = data
  const hasActiveLiens = encumbrances.active_liens && encumbrances.active_liens.length > 0

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <h1 className={styles.title}>Parcel {parcel.cadastral_number}</h1>
        <Badge variant={hasActiveLiens ? 'active' : 'clean'}>
          {hasActiveLiens ? 'Encumbered' : 'Clean'}
        </Badge>
      </div>

      <div className={styles.metrics}>
        <MetricCard
          label="Credit Score"
          value={credit_intelligence ? String(credit_intelligence.ai_score) : 'N/A'}
        />
        <MetricCard
          label="Active Liens"
          value={String(encumbrances.active_liens?.length ?? 0)}
        />
        <MetricCard label="Area (ha)" value={String(parcel.area_ha)} />
      </div>

      <Card padding="lg">
        <h2 className={styles.sectionTitle}>Parcel Details</h2>
        <div className={styles.details}>
          <div className={styles.detailLabel}>Owner Wallet</div>
          <div className={styles.detailValue}>{parcel.owner_wallet}</div>
          <div className={styles.detailLabel}>Oblast / Rayon</div>
          <div className={styles.detailValue}>{parcel.oblast ?? '-'} / {parcel.rayon ?? '-'}</div>
          <div className={styles.detailLabel}>Land Class</div>
          <div className={styles.detailValue}>{parcel.land_class}</div>
          <div className={styles.detailLabel}>NDVI Trend</div>
          <div className={styles.detailValue}>{productivity.ndvi_trend}</div>
          {parcel.on_chain_address && (
            <>
              <div className={styles.detailLabel}>On-Chain</div>
              <div className={styles.detailValue}>
                <a
                  href={`https://explorer.solana.com/address/${parcel.on_chain_address}?cluster=devnet`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className={styles.explorerLink}
                >
                  {parcel.on_chain_address.slice(0, 8)}...{parcel.on_chain_address.slice(-8)}
                </a>
              </div>
            </>
          )}
        </div>
      </Card>

      <Card padding="lg">
        <h2 className={styles.sectionTitle}>NDVI History</h2>
        <NDVIChart certificates={productivity.certificates ?? []} />
      </Card>

      {credit_intelligence && (
        <Card padding="lg">
          <h2 className={styles.sectionTitle}>Credit Intelligence</h2>
          <div className={styles.creditSection}>
            <CreditGauge
              score={credit_intelligence.ai_score}
              grade={credit_intelligence.collateral_grade}
              ltv={credit_intelligence.recommended_ltv}
            />
            <div className={styles.creditDetails}>
              <p>{credit_intelligence.explanation}</p>
              {credit_intelligence.risk_factors && credit_intelligence.risk_factors.length > 0 && (
                <div className={styles.riskFactors}>
                  <strong>Risk Factors:</strong>
                  <ul>
                    {credit_intelligence.risk_factors.map((f, i) => (
                      <li key={i} className={styles.riskFactor}>{f}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          </div>
        </Card>
      )}

      <Card padding="lg">
        <h2 className={styles.sectionTitle}>Lien History</h2>
        {encumbrances.active_liens && encumbrances.active_liens.length > 0 ? (
          <div className={styles.lienList}>
            {encumbrances.active_liens.map((lien) => (
              <div key={lien.id} className={styles.lienItem}>
                <div>
                  <Badge variant={lien.status === 'active' ? 'active' : 'released'}>
                    {lien.status}
                  </Badge>
                  <span className={styles.lienAmount}>
                    {lien.amount_tenge.toLocaleString()} KZT
                  </span>
                </div>
                <div className={styles.lienMeta}>
                  {lien.lender_wallet.slice(0, 8)}... | {new Date(lien.registered_at).toLocaleDateString()}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className={styles.placeholder}>
            <p className={styles.placeholderTitle}>No liens recorded</p>
            <p className={styles.placeholderHint}>This parcel has no active encumbrances</p>
          </div>
        )}
        <div className={styles.lienMeta}>
          Total historical liens: {encumbrances.lien_count_historical}
        </div>
      </Card>
    </div>
  )
}
