import { useState } from 'react'
import type { Address } from '@solana/kit'
import { useSignAndSend } from '../../../solana/transaction'
import { buildRegisterEncumbranceInstruction, getParcelPda } from '../../../solana/program'
import { useLien } from '../../../hooks/useLien'
import { useToast } from '../../../components/Toast/Toast'
import { Button } from '../../../components/Button/Button'
import { Input } from '../../../components/Input/Input'
import styles from '../WizardFlow.module.css'

interface Props {
  cadastral: string
  onBack: () => void
  onDone: () => void
}

export function RegisterLienStep({ cadastral, onBack, onDone }: Props) {
  const { signAndSend, connected, walletAddress, txStatus } = useSignAndSend()
  const { registerLien } = useLien()
  const { toast } = useToast()

  const [amount, setAmount] = useState('')
  const [notaryHash, setNotaryHash] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  if (!connected) {
    return (
      <div className={styles.stepCenter}>
        <h2 className={styles.stepTitle}>Register Lien</h2>
        <p className={styles.stepDesc}>
          Connect your wallet to register a lien on this parcel.
        </p>
        <p className={styles.stepHint}>
          Click <strong>Connect Wallet</strong> in the top right corner to continue.
        </p>
      </div>
    )
  }

  if (success) {
    return (
      <div className={styles.stepCenter}>
        <h2 className={styles.stepTitle}>Lien Registered</h2>
        <div className={styles.ndviResult}>
          <div className={styles.ndviScore} style={{ color: 'var(--color-primary)', fontSize: '48px' }}>✓</div>
          <div className={styles.ndviLabel}>On-Chain Encumbrance</div>
          <div className={styles.ndviDesc}>
            <strong>{Number(amount).toLocaleString()} KZT</strong> lien registered on parcel <strong>{cadastral}</strong>
          </div>
        </div>
        <p className={styles.stepDesc}>
          The encumbrance has been recorded on Solana and in the TerraLedger database.
          This parcel is now marked as encumbered in credit reports.
        </p>
        <div className={styles.stepActions}>
          <Button variant="secondary" onClick={onBack}>Back to Profile</Button>
          <Button onClick={onDone}>Search Another Parcel</Button>
        </div>
      </div>
    )
  }

  const handleSubmit = async () => {
    const amountNum = Number(amount)
    if (!amountNum || amountNum <= 0) {
      setError('Please enter a valid amount')
      return
    }

    setError(null)
    setSubmitting(true)

    try {
      const [parcelPda] = await getParcelPda(cadastral)
      const notarySigHash = new Uint8Array(32)
      const notaryCertHash = new Uint8Array(32)

      if (notaryHash) {
        const encoder = new TextEncoder()
        const hashBytes = encoder.encode(notaryHash)
        notaryCertHash.set(hashBytes.slice(0, 32))
      }

      const ix = await buildRegisterEncumbranceInstruction(
        walletAddress as Address,
        parcelPda,
        cadastral,
        BigInt(amountNum),
        notarySigHash,
        notaryCertHash,
      )
      await signAndSend([ix])

      await registerLien({
        cadastral_number: cadastral,
        lender_wallet: String(walletAddress),
        amount_tenge: amountNum,
        notary_cert_hash: notaryHash,
      })

      setSuccess(true)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to register lien'
      setError(msg)
      toast(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className={styles.stepCenter}>
      <h2 className={styles.stepTitle}>Register Lien</h2>
      <p className={styles.stepDesc}>
        Register an encumbrance on parcel <strong>{cadastral}</strong>.
      </p>

      <div className={styles.stepForm}>
        <div className={styles.formFields}>
          <Input
            label="Amount (KZT)"
            type="number"
            placeholder="e.g. 5000000"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          <Input
            label="Notary Hash (optional)"
            placeholder="Notary certificate hash"
            value={notaryHash}
            onChange={(e) => setNotaryHash(e.target.value)}
          />
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
            Register Lien
          </Button>
        </div>
      </div>
    </div>
  )
}
