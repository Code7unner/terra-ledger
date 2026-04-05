import { useState, useEffect, useCallback } from 'react'
import { Card } from '../../components/Card/Card'
import { Badge } from '../../components/Badge/Badge'
import { Button } from '../../components/Button/Button'
import { useSignAndSend } from '../../solana/transaction'
import { get, post } from '../../api/client'
import styles from './ConsentDashboard.module.css'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface Consent {
  wallet_address: string
  status: string
  granted_at?: string
  revoked_at?: string
}

interface ConsentLogEntry {
  lender_name: string
  data_type: string
  accessed_at: string
}

type BadgeVariant = 'active' | 'released' | 'warning'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const SHARED_DATA_TYPES = [
  { name: 'Parcel Info', description: 'Cadastral number, area, land class, and location' },
  { name: 'NDVI Certificates', description: 'Seasonal satellite-verified productivity scores' },
  { name: 'Lien Status', description: 'Active encumbrances and pledge history' },
  { name: 'Credit Score', description: 'AI-generated creditworthiness assessment' },
]

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function consentBadgeVariant(status: string): BadgeVariant {
  if (status === 'granted') return 'active'
  if (status === 'revoked') return 'released'
  return 'warning'
}

function consentBadgeLabel(status: string): string {
  if (status === 'granted') return 'Granted'
  if (status === 'revoked') return 'Revoked'
  return 'Pending'
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function ConsentDashboard() {
  const { connected, walletAddress } = useSignAndSend()

  const [consent, setConsent] = useState<Consent | null>(null)
  const [accessLog, setAccessLog] = useState<ConsentLogEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [actionLoading, setActionLoading] = useState(false)

  const walletStr = walletAddress ? String(walletAddress) : ''

  const fetchConsentData = useCallback(async () => {
    if (!walletStr) return
    setLoading(true)
    try {
      const [consentData, logData] = await Promise.all([
        get<Consent>(`/api/v1/consent/${encodeURIComponent(walletStr)}`).catch(() => null),
        get<ConsentLogEntry[]>(`/api/v1/consent/${encodeURIComponent(walletStr)}/log`).catch(() => []),
      ])
      setConsent(consentData)
      setAccessLog(logData ?? [])
    } finally {
      setLoading(false)
    }
  }, [walletStr])

  useEffect(() => {
    if (connected && walletStr) {
      fetchConsentData()
    }
  }, [connected, walletStr, fetchConsentData])

  async function handleGrant() {
    if (!walletStr) return
    setActionLoading(true)
    try {
      await post<Consent>('/api/v1/consent/grant', { wallet_address: walletStr })
      await fetchConsentData()
    } finally {
      setActionLoading(false)
    }
  }

  async function handleRevoke() {
    if (!walletStr) return
    setActionLoading(true)
    try {
      await post<Consent>('/api/v1/consent/revoke', { wallet_address: walletStr })
      await fetchConsentData()
    } finally {
      setActionLoading(false)
    }
  }

  // Wallet not connected state
  if (!connected || !walletAddress) {
    return (
      <div className={styles.page}>
        <h1 className={styles.title}>Consent Dashboard</h1>
        <p className={styles.subtitle}>PDPA consent tracking and management</p>
        <Card padding="lg">
          <div className={styles.walletRequired}>
            <div className={styles.walletIcon} aria-hidden="true">🔗</div>
            <h2 className={styles.sectionTitle}>Wallet Required</h2>
            <p>Connect your Solana wallet to manage data processing consent for your parcels.</p>
            <p className={styles.walletHint}>
              Your consent preferences are stored on-chain and can be revoked at any time.
            </p>
          </div>
        </Card>
      </div>
    )
  }

  const consentStatus = consent?.status ?? 'pending'
  const isGranted = consentStatus === 'granted'

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>Consent Dashboard</h1>
      <p className={styles.subtitle}>PDPA consent tracking and management</p>

      {/* Consent Status Card */}
      <Card padding="lg">
        <div className={styles.consentStatus}>
          <div className={styles.consentHeader}>
            <h2 className={styles.sectionTitle}>Data Processing Consent</h2>
            {loading ? (
              <Badge variant="warning">Loading...</Badge>
            ) : (
              <Badge variant={consentBadgeVariant(consentStatus)}>
                {consentBadgeLabel(consentStatus)}
              </Badge>
            )}
          </div>
          <p className={styles.consentDescription}>
            Grant consent for your parcel and credit data to be processed by lenders
            on the TerraLedger platform. You can revoke consent at any time.
          </p>
          <div className={styles.actions}>
            <Button
              variant="primary"
              onClick={handleGrant}
              disabled={actionLoading || isGranted}
              loading={actionLoading}
            >
              Grant Consent
            </Button>
            <Button
              variant="secondary"
              onClick={handleRevoke}
              disabled={actionLoading || !isGranted}
              loading={actionLoading}
            >
              Revoke Consent
            </Button>
          </div>
        </div>
      </Card>

      {/* Shared Data Types */}
      <Card padding="md">
        <h2 className={styles.sectionTitle}>Shared Data Types</h2>
        <div className={styles.dataTypes}>
          {SHARED_DATA_TYPES.map((dt) => (
            <div key={dt.name} className={styles.dataTypeItem}>
              <span className={styles.dataTypeName}>{dt.name}</span>
              <span className={styles.dataTypeDesc}>{dt.description}</span>
            </div>
          ))}
        </div>
      </Card>

      {/* Access Log */}
      <Card padding="md">
        <h2 className={styles.sectionTitle}>Access Log</h2>
        {accessLog.length > 0 ? (
          <table className={styles.accessLogTable}>
            <thead>
              <tr>
                <th>Lender Name</th>
                <th>Data Type</th>
                <th>Accessed At</th>
              </tr>
            </thead>
            <tbody>
              {accessLog.map((entry, i) => (
                <tr key={i} className={styles.logRow}>
                  <td>{entry.lender_name}</td>
                  <td>{entry.data_type}</td>
                  <td>{new Date(entry.accessed_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <div className={styles.placeholder}>
            <p className={styles.placeholderTitle}>
              {loading ? 'Loading access log...' : 'No access log entries'}
            </p>
            {!loading && (
              <p className={styles.placeholderHint}>
                When lenders access your data, each request will appear here
              </p>
            )}
          </div>
        )}
      </Card>
    </div>
  )
}
