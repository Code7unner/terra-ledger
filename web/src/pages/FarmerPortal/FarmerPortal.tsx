import { useState, useEffect, useCallback } from 'react'
import { type Address } from '@solana/kit'
import { Card } from '../../components/Card/Card'
import { Button } from '../../components/Button/Button'
import { Input } from '../../components/Input/Input'
import { NDVIChart } from '../../components/NDVIChart/NDVIChart'
import { useParcel } from '../../hooks/useParcel'
import type { RegisterParcelInput } from '../../hooks/useParcel'
import { useSignAndSend } from '../../solana/transaction'
import { buildRegisterParcelInstruction } from '../../solana/program'
import { get } from '../../api/client'
import styles from './FarmerPortal.module.css'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface Certificate {
  season: string
  ndvi_score: number
  created_at: string
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const TX_STATUS_LABELS: Record<string, string> = {
  idle: '',
  building: 'Building transaction...',
  signing: 'Waiting for wallet signature...',
  confirming: 'Confirming on-chain...',
  done: 'Transaction confirmed!',
  error: 'Transaction failed',
}

function explorerUrl(sig: string): string {
  return `https://explorer.solana.com/tx/${sig}?cluster=devnet`
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function FarmerPortal() {
  const { data: registeredParcel, loading, error, registerParcel } = useParcel()
  const {
    signAndSend,
    txStatus,
    txSignature,
    txError,
    connected,
    walletAddress,
    reset: resetTx,
  } = useSignAndSend()

  const [form, setForm] = useState<RegisterParcelInput>({
    cadastral_number: '',
    owner_wallet: '',
    area_ha: 0,
    land_class: 1,
    oblast: '',
    rayon: '',
    holder_name: '',
    holder_iin_hash: '',
  })

  const [certificates, setCertificates] = useState<Certificate[]>([])
  const [certsLoading, setCertsLoading] = useState(false)

  // Auto-fill owner_wallet when wallet connects
  useEffect(() => {
    if (connected && walletAddress && !form.owner_wallet) {
      setForm((prev) => ({ ...prev, owner_wallet: String(walletAddress) }))
    }
  }, [connected, walletAddress, form.owner_wallet])

  function update(field: keyof RegisterParcelInput, value: string | number) {
    setForm((prev) => ({ ...prev, [field]: value }))
  }

  // Fetch certificates for a given cadastral number
  const fetchCertificates = useCallback(async (cadastral: string) => {
    if (!cadastral) return
    setCertsLoading(true)
    try {
      const certs = await get<Certificate[]>(
        `/api/v1/parcels/${encodeURIComponent(cadastral)}/certificates`,
      )
      setCertificates(certs ?? [])
    } catch {
      setCertificates([])
    } finally {
      setCertsLoading(false)
    }
  }, [])

  async function handleSubmit() {
    if (!form.cadastral_number || !form.owner_wallet) return

    resetTx()

    // If wallet is connected, build and send on-chain transaction first
    if (connected && walletAddress) {
      try {
        const egissHash = new Uint8Array(32) // mock EGISS hash
        const ix = await buildRegisterParcelInstruction(
          walletAddress as Address,
          form.cadastral_number,
          form.area_ha,
          form.land_class,
          egissHash,
        )
        await signAndSend([ix])
      } catch {
        // txError is already set by the hook; do not block REST call
        return
      }
    }

    // Also register via REST API
    const result = await registerParcel(form)

    // Fetch certificates after successful registration
    if (result) {
      await fetchCertificates(form.cadastral_number)
    }
  }

  // Re-fetch certificates when a parcel is already registered
  useEffect(() => {
    if (registeredParcel?.cadastral_number) {
      fetchCertificates(registeredParcel.cadastral_number)
    }
  }, [registeredParcel?.cadastral_number, fetchCertificates])

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>Farmer Portal</h1>
      <p className={styles.subtitle}>Register and manage your parcels</p>

      {/* Wallet info bar */}
      {connected && walletAddress && (
        <div className={styles.walletInfo}>
          Connected: {String(walletAddress).slice(0, 8)}...{String(walletAddress).slice(-4)}
        </div>
      )}

      <Card padding="lg">
        <h2 className={styles.sectionTitle}>Register Parcel</h2>
        <div className={styles.form}>
          <Input
            label="Cadastral Number"
            placeholder="KZ11-0032-001"
            value={form.cadastral_number}
            onChange={(e) => update('cadastral_number', e.target.value)}
          />
          <Input
            label="Owner Wallet"
            placeholder="Solana wallet address"
            value={form.owner_wallet}
            onChange={(e) => update('owner_wallet', e.target.value)}
          />
          <Input
            label="Area (hectares)"
            placeholder="0.00"
            type="number"
            suffix="ha"
            value={form.area_ha || ''}
            onChange={(e) => update('area_ha', Number(e.target.value))}
          />
          <Input
            label="Land Class (1-5)"
            placeholder="1"
            type="number"
            value={form.land_class || ''}
            onChange={(e) => update('land_class', Number(e.target.value))}
          />
          <Input
            label="Oblast"
            placeholder="Akmola"
            value={form.oblast}
            onChange={(e) => update('oblast', e.target.value)}
          />
          <Input
            label="Rayon"
            placeholder="Shortandy"
            value={form.rayon}
            onChange={(e) => update('rayon', e.target.value)}
          />
          <Input
            label="Holder Name"
            placeholder="Full name"
            value={form.holder_name}
            onChange={(e) => update('holder_name', e.target.value)}
          />
          <Button
            variant="primary"
            fullWidth
            onClick={handleSubmit}
            disabled={loading || (txStatus !== 'idle' && txStatus !== 'done' && txStatus !== 'error')}
          >
            {loading ? 'Registering...' : connected ? 'Sign & Register Parcel' : 'Register Parcel'}
          </Button>

          {/* Transaction progress */}
          {txStatus !== 'idle' && (
            <div className={styles.txProgress} aria-live="polite">
              <span>{TX_STATUS_LABELS[txStatus]}</span>
              {txStatus === 'done' && txSignature && (
                <a
                  className={styles.explorerLink}
                  href={explorerUrl(txSignature)}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  View on Explorer
                </a>
              )}
              {txStatus === 'error' && txError && (
                <span className={styles.errorText}>{txError}</span>
              )}
            </div>
          )}

          {error && <div className={styles.errorText} role="alert">{error}</div>}
        </div>
      </Card>

      <Card padding="md">
        <h2 className={styles.sectionTitle}>My Parcels</h2>
        {registeredParcel ? (
          <div className={styles.successCard}>
            <h3>Parcel Registered</h3>
            <div className={styles.parcelInfo}>
              <div className={styles.infoRow}>
                <span className={styles.infoLabel}>Cadastral</span>
                <span className={styles.infoValue}>{registeredParcel.cadastral_number}</span>
              </div>
              <div className={styles.infoRow}>
                <span className={styles.infoLabel}>Area</span>
                <span className={styles.infoValue}>{registeredParcel.area_ha} ha</span>
              </div>
              <div className={styles.infoRow}>
                <span className={styles.infoLabel}>Oblast</span>
                <span className={styles.infoValue}>{registeredParcel.oblast ?? '-'}</span>
              </div>
              <div className={styles.infoRow}>
                <span className={styles.infoLabel}>Owner</span>
                <span className={styles.infoValue}>
                  {registeredParcel.owner_wallet.slice(0, 12)}...
                </span>
              </div>
            </div>
          </div>
        ) : (
          <div className={styles.placeholder}>
            <p className={styles.placeholderTitle}>No parcels registered</p>
            <p className={styles.placeholderHint}>
              Fill in the form above to register your first parcel on-chain
            </p>
          </div>
        )}
      </Card>

      {/* Certificate History */}
      {registeredParcel && (
        <Card padding="md">
          <h2 className={styles.sectionTitle}>Certificate History</h2>
          <div className={styles.certHistory}>
            {certsLoading ? (
              <div className={styles.placeholder}>Loading certificates...</div>
            ) : certificates.length > 0 ? (
              <>
                <NDVIChart certificates={certificates} />
                <ul className={styles.certList}>
                  {certificates.map((cert, i) => (
                    <li key={i} className={styles.certItem}>
                      <span className={styles.certSeason}>{cert.season}</span>
                      <span className={styles.certScore}>
                        NDVI: {cert.ndvi_score.toFixed(2)}
                      </span>
                      <span className={styles.certDate}>
                        {new Date(cert.created_at).toLocaleDateString()}
                      </span>
                    </li>
                  ))}
                </ul>
              </>
            ) : (
              <div className={styles.placeholder}>
                No certificates yet. Certificates are issued after seasonal NDVI checks.
              </div>
            )}
          </div>
        </Card>
      )}
    </div>
  )
}
