import { useState, useEffect } from 'react'
import type { Address } from '@solana/kit'
import type { WizardData } from '../WizardFlow'
import { useSignAndSend } from '../../../solana/transaction'
import { buildRegisterParcelInstruction } from '../../../solana/program'
import { useParcel } from '../../../hooks/useParcel'
import { useToast } from '../../../components/Toast/Toast'
import { Button } from '../../../components/Button/Button'
import { Input } from '../../../components/Input/Input'
import styles from '../WizardFlow.module.css'

interface Props {
  data: WizardData
  isDemo: boolean
  onUpdate: (partial: Partial<WizardData>) => void
  onNext: () => void
  onBack: () => void
}

const OBLAST_OPTIONS = [
  'Akmola',
  'Aktobe',
  'Almaty',
  'Atyrau',
  'East Kazakhstan',
  'Karaganda',
  'Kostanay',
  'Kyzylorda',
  'Mangystau',
  'North Kazakhstan',
  'Pavlodar',
  'South Kazakhstan',
  'Turkestan',
  'West Kazakhstan',
  'Zhambyl',
]

const LAND_CLASS_OPTIONS = [
  { value: 1, label: '1 - Highest productivity (irrigated, fertile)' },
  { value: 2, label: '2 - Good (rainfed, productive)' },
  { value: 3, label: '3 - Average' },
  { value: 4, label: '4 - Low (pasture, degraded)' },
  { value: 5, label: '5 - Lowest (desert, unused)' },
]

export function RegisterParcelStep({ data, isDemo, onUpdate, onNext, onBack }: Props) {
  const { signAndSend, connected, walletAddress, txStatus } = useSignAndSend()
  const { registerParcel } = useParcel()
  const { toast } = useToast()

  const [cadastral, setCadastral] = useState(data.cadastral)
  const [areaHa, setAreaHa] = useState(data.area_ha || 0)
  const [landClass, setLandClass] = useState(data.land_class || 2)
  const [oblast, setOblast] = useState(data.oblast)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (isDemo && !cadastral) {
      setCadastral('KZ11-0032-001')
      setAreaHa(150)
      setLandClass(2)
      setOblast('Akmola')
    }
  }, [isDemo, cadastral])

  const handleSubmit = async () => {
    if (!cadastral || !areaHa || !oblast) {
      setError('Please fill in all required fields')
      return
    }

    setError(null)
    setSubmitting(true)

    try {
      if (connected && walletAddress) {
        const egissHash = new Uint8Array(32)
        const ix = await buildRegisterParcelInstruction(
          walletAddress as Address,
          cadastral,
          areaHa,
          landClass,
          egissHash,
        )
        await signAndSend([ix])
      }

      await registerParcel({
        cadastral_number: cadastral,
        owner_wallet: walletAddress ? String(walletAddress) : 'demo-wallet',
        area_ha: areaHa,
        land_class: landClass,
        oblast,
        rayon: '',
        holder_name: '',
        holder_iin_hash: '',
      })

      onUpdate({
        cadastral,
        area_ha: areaHa,
        land_class: landClass,
        oblast,
      })
      onNext()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to register parcel'
      setError(msg)
      toast(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className={styles.stepCenter}>
      <h2 className={styles.stepTitle}>Register Your Parcel</h2>
      <p className={styles.stepDesc}>
        Enter the details of your land parcel to register it on the blockchain.
      </p>

      <div className={styles.stepForm}>
        <div className={styles.formFields}>
          <Input
            label="Cadastral Number"
            placeholder="e.g. KZ11-0032-001"
            value={cadastral}
            onChange={(e) => setCadastral(e.target.value)}
          />

          <Input
            label="Area"
            type="number"
            placeholder="e.g. 150"
            suffix="ha"
            value={areaHa || ''}
            onChange={(e) => setAreaHa(Number(e.target.value))}
          />

          <div className={styles.fieldGroup}>
            <label className={styles.fieldLabel}>Land Class</label>
            <select
              className={styles.select}
              value={landClass}
              onChange={(e) => setLandClass(Number(e.target.value))}
            >
              {LAND_CLASS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          <div className={styles.fieldGroup}>
            <label className={styles.fieldLabel}>Oblast</label>
            <select
              className={styles.select}
              value={oblast}
              onChange={(e) => setOblast(e.target.value)}
            >
              <option value="">Select oblast...</option>
              {OBLAST_OPTIONS.map((name) => (
                <option key={name} value={name}>
                  {name}
                </option>
              ))}
            </select>
          </div>
        </div>

        {error && <p className={styles.error}>{error}</p>}

        <div className={styles.stepActions}>
          <Button variant="secondary" onClick={onBack}>
            Back
          </Button>
          <Button
            loading={submitting || txStatus === 'signing' || txStatus === 'confirming'}
            onClick={handleSubmit}
          >
            Register Parcel
          </Button>
        </div>
      </div>
    </div>
  )
}
