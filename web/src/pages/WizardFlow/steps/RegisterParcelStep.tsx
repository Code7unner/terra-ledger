import { useState, useEffect } from 'react'
import type { Address } from '@solana/kit'
import type { WizardData } from '../WizardFlow'
import { useSignAndSend } from '../../../solana/transaction'
import { buildRegisterParcelInstruction } from '../../../solana/program'
import { useParcel } from '../../../hooks/useParcel'
import { post } from '../../../api/client'
import { useToast } from '../../../components/Toast/useToast'
import { Button } from '../../../components/Button/Button'
import { Input } from '../../../components/Input/Input'
import styles from '../WizardFlow.module.css'

interface Props {
  data: WizardData
  isDemo: boolean
  onUpdate: (partial: Partial<WizardData>) => void
  onNext: () => void
  onBack: () => void
  onSkipToSummary?: () => void
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

export function RegisterParcelStep({ data, isDemo, onUpdate, onNext, onBack, onSkipToSummary }: Props) {
  const { signAndSend, connected, walletAddress: extensionWallet, txStatus } = useSignAndSend()
  const { registerParcel } = useParcel()
  const { toast } = useToast()

  const walletAddress = extensionWallet ? String(extensionWallet) : data.walletAddress

  const [cadastral, setCadastral] = useState(data.cadastral)
  const [areaHa, setAreaHa] = useState(data.area_ha || 0)
  const [landClass, setLandClass] = useState(data.land_class || 2)
  const [oblast, setOblast] = useState(data.oblast)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (isDemo && !cadastral) {
      setCadastral('KZ11-0033-001')
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

    // Update wizard data immediately so downstream steps have cadastral
    onUpdate({
      cadastral,
      area_ha: areaHa,
      land_class: landClass,
      oblast,
    })

    try {
      let alreadyOnChain = false

      if (connected && walletAddress) {
        try {
          const egissHash = new Uint8Array(32)
          const ix = await buildRegisterParcelInstruction(
            walletAddress as Address,
            cadastral,
            areaHa,
            landClass,
            egissHash,
          )
          await signAndSend([ix])
        } catch (txErr: unknown) {
          const msg = txErr instanceof Error ? txErr.message : String(txErr)
          const isAlreadyRegistered =
            msg.includes('already in use') ||
            msg.includes('custom program error: 0x0') ||
            msg.includes('code 0') ||
            msg.includes('Code: 0')
          if (!isAlreadyRegistered) {
            throw txErr
          }
          alreadyOnChain = true
        }
      }

      if (alreadyOnChain) {
        toast('Parcel already registered on-chain. Loading profile...', 'success')
        setSubmitting(false)
        if (onSkipToSummary) {
          onSkipToSummary()
        } else {
          onNext()
        }
        return
      }

      await registerParcel({
        cadastral_number: cadastral,
        owner_wallet: walletAddress || 'demo-wallet',
        area_ha: areaHa,
        land_class: landClass,
        oblast,
        rayon: '',
        holder_name: '',
        holder_iin_hash: '',
      })

      // Auto-grant consent so credit scoring works
      if (walletAddress) {
        post('/api/v1/consent/grant', { wallet_address: String(walletAddress) }).catch(() => {})
      }

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
            placeholder="e.g. KZ11-0033-001"
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
