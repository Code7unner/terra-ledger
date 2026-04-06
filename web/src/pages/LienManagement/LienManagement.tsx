import { useState } from 'react'
import { type Address, address } from '@solana/kit'
import { Card } from '../../components/Card/Card'
import { Button } from '../../components/Button/Button'
import { Input } from '../../components/Input/Input'
import { Badge } from '../../components/Badge/Badge'
import { useLien } from '../../hooks/useLien'
import {
  buildRegisterEncumbranceInstruction,
  buildReleaseEncumbranceInstruction,
  getParcelPda,
} from '../../solana/program'
import { useSignAndSend, type OnChainTxStatus } from '../../solana/transaction'
import styles from './LienManagement.module.css'

type RestTxStatus = 'idle' | 'submitting' | 'success' | 'error'

const TX_STATUS_LABELS: Record<OnChainTxStatus, string> = {
  idle: '',
  building: 'Building transaction...',
  signing: 'Waiting for wallet signature...',
  confirming: 'Confirming on-chain...',
  done: 'Transaction confirmed',
  error: 'Transaction failed',
}

function explorerUrl(sig: string): string {
  return `https://explorer.solana.com/tx/${sig}?cluster=devnet`
}

export default function LienManagement() {
  const { liens, loading, error, fetchLiens, registerLien, releaseLien } = useLien()

  // Separate hook instances for register and release so their state is independent
  const {
    signAndSend: registerSignAndSend,
    txStatus: registerTxStatus,
    txSignature: registerTxSignature,
    txError: registerTxError,
    reset: resetRegisterTx,
    connected: walletConnected,
    walletAddress,
  } = useSignAndSend()

  const {
    signAndSend: releaseSignAndSend,
    txStatus: releaseTxStatus,
    txSignature: releaseTxSignature,
    txError: releaseTxError,
    reset: resetReleaseTx,
  } = useSignAndSend()

  // --- Register form state ---
  const [cadastral, setCadastral] = useState('')
  const [amount, setAmount] = useState('')
  const [wallet, setWallet] = useState(() => walletAddress || '')
  const [notaryHash, setNotaryHash] = useState('')
  const [registerStatus, setRegisterStatus] = useState<RestTxStatus>('idle')

  // --- Release form state ---
  const [releaseId, setReleaseId] = useState('')
  const [releaseStatus, setReleaseStatus] = useState<RestTxStatus>('idle')

  // --- Search state ---
  const [searchCadastral, setSearchCadastral] = useState('')

  async function handleRegister() {
    if (!cadastral || !amount || !wallet) return

    // Reset previous state
    resetRegisterTx()
    setRegisterStatus('idle')

    if (!walletConnected) {
      setRegisterStatus('error')
      return
    }

    try {
      // Step 1: Build instruction
      const lender: Address = address(wallet)
      const [parcelPda] = await getParcelPda(cadastral)

      // Use zero-filled hash placeholders (32 bytes each)
      // In production these come from notary GOST signing
      const notarySigHash = new Uint8Array(32)
      const notaryCertHash = new Uint8Array(32)

      // If a notary hash was provided, use it for the cert hash
      if (notaryHash) {
        const encoder = new TextEncoder()
        const hashBytes = encoder.encode(notaryHash)
        notaryCertHash.set(hashBytes.slice(0, 32))
      }

      const amountLamports = BigInt(amount)

      const ix = await buildRegisterEncumbranceInstruction(
        lender,
        parcelPda,
        cadastral,
        amountLamports,
        notarySigHash,
        notaryCertHash,
      )

      // Step 2: Sign and send on-chain
      await registerSignAndSend([ix])

      // Step 3: Also POST to REST API for PostgreSQL record (belt-and-suspenders)
      setRegisterStatus('submitting')
      const result = await registerLien({
        cadastral_number: cadastral,
        lender_wallet: wallet,
        amount_tenge: Number(amount),
        notary_cert_hash: notaryHash,
      })
      setRegisterStatus(result ? 'success' : 'error')
    } catch {
      // txError is already set by useSignAndSend if the on-chain part failed
      if (registerTxStatus !== 'error') {
        setRegisterStatus('error')
      }
    }
  }

  async function handleRelease(id: string) {
    setReleaseStatus('submitting')
    resetReleaseTx()

    // Attempt on-chain release if wallet is connected and we have
    // cadastral info from the lien list
    if (walletConnected && walletAddress && searchCadastral) {
      try {
        const lender: Address = address(walletAddress)
        const [parcelPda] = await getParcelPda(searchCadastral)

        const ix = await buildReleaseEncumbranceInstruction(lender, parcelPda)

        await releaseSignAndSend([ix])
      } catch {
        // releaseTxError is set by the hook
      }
    }

    // Also release via REST API
    await releaseLien(id)
    setReleaseStatus(error ? 'error' : 'success')
  }

  async function handleSearch() {
    if (!searchCadastral) return
    await fetchLiens(searchCadastral)
  }

  const isRegistering = registerTxStatus === 'building' || registerTxStatus === 'signing' || registerTxStatus === 'confirming'

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>Lien Management</h1>
      <p className={styles.subtitle}>Register and release liens on parcels</p>

      {!walletConnected && (
        <div className={styles.walletWarning} role="alert">
          Connect your wallet to sign on-chain transactions.
          REST-only mode is available but liens will not be recorded on Solana.
        </div>
      )}

      <Card padding="lg">
        <h2 className={styles.sectionTitle}>Register Lien</h2>
        <div className={styles.form}>
          <Input
            label="Cadastral Number"
            placeholder="KZ11-0033-001"
            value={cadastral}
            onChange={(e) => setCadastral(e.target.value)}
          />
          <Input
            label="Lender Wallet"
            placeholder="Solana wallet address"
            value={wallet}
            onChange={(e) => setWallet(e.target.value)}
          />
          <Input
            label="Amount (KZT)"
            placeholder="0"
            type="number"
            suffix="KZT"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          <Input
            label="Notary Certificate Hash"
            placeholder="GOST hash..."
            value={notaryHash}
            onChange={(e) => setNotaryHash(e.target.value)}
          />
          <Button
            variant="primary"
            fullWidth
            onClick={handleRegister}
            disabled={loading || isRegistering}
            loading={isRegistering}
          >
            {walletConnected ? 'Register Lien (On-Chain)' : 'Register Lien'}
          </Button>

          {registerTxStatus !== 'idle' && registerTxStatus !== 'done' && registerTxStatus !== 'error' && (
            <div className={styles.txProgress} aria-live="polite">
              {TX_STATUS_LABELS[registerTxStatus]}
            </div>
          )}

          {registerTxStatus === 'done' && registerTxSignature && (
            <div className={styles.statusSuccess} role="status">
              On-chain transaction confirmed.{' '}
              <a
                href={explorerUrl(registerTxSignature)}
                target="_blank"
                rel="noopener noreferrer"
                className={styles.explorerLink}
              >
                View on Explorer
              </a>
            </div>
          )}

          {registerTxStatus === 'error' && (
            <div className={styles.statusError} role="alert">{registerTxError || 'On-chain transaction failed'}</div>
          )}

          {registerStatus === 'success' && (
            <div className={styles.statusSuccess}>Lien registered in database</div>
          )}
          {registerStatus === 'error' && registerTxStatus !== 'error' && (
            <div className={styles.statusError}>{error || 'REST registration failed'}</div>
          )}
        </div>
      </Card>

      <Card padding="lg">
        <h2 className={styles.sectionTitle}>Release Lien</h2>
        <div className={styles.form}>
          <Input
            label="Lien ID"
            placeholder="Enter lien ID..."
            value={releaseId}
            onChange={(e) => setReleaseId(e.target.value)}
          />
          <Button
            variant="danger"
            fullWidth
            onClick={() => handleRelease(releaseId)}
            disabled={loading || !releaseId}
          >
            {releaseStatus === 'submitting' ? 'Releasing...' : 'Release Lien'}
          </Button>

          {releaseTxStatus !== 'idle' && releaseTxStatus !== 'done' && releaseTxStatus !== 'error' && (
            <div className={styles.txProgress} aria-live="polite">
              {TX_STATUS_LABELS[releaseTxStatus]}
            </div>
          )}

          {releaseTxStatus === 'done' && releaseTxSignature && (
            <div className={styles.statusSuccess} role="status">
              On-chain release confirmed.{' '}
              <a
                href={explorerUrl(releaseTxSignature)}
                target="_blank"
                rel="noopener noreferrer"
                className={styles.explorerLink}
              >
                View on Explorer
              </a>
            </div>
          )}

          {releaseTxStatus === 'error' && (
            <div className={styles.statusError} role="alert">{releaseTxError || 'On-chain release failed'}</div>
          )}

          {releaseStatus === 'success' && (
            <div className={styles.statusSuccess}>Lien released</div>
          )}
        </div>
      </Card>

      <Card padding="md">
        <h2 className={styles.sectionTitle}>Active Liens</h2>
        <div className={styles.searchRow}>
          <Input
            placeholder="Cadastral number..."
            value={searchCadastral}
            onChange={(e) => setSearchCadastral(e.target.value)}
          />
          <Button variant="primary" onClick={handleSearch} disabled={loading}>
            Load
          </Button>
        </div>

        {liens.length > 0 ? (
          <div className={styles.lienList}>
            {liens.map((lien) => (
              <div key={lien.id} className={styles.lienItem}>
                <div className={styles.lienInfo}>
                  <Badge variant={lien.status === 'active' ? 'active' : 'released'}>
                    {lien.status}
                  </Badge>
                  <span className={styles.lienAmount}>
                    {lien.amount_tenge.toLocaleString()} KZT
                  </span>
                  <span className={styles.lienWallet}>
                    {lien.lender_wallet.slice(0, 8)}...
                  </span>
                </div>
                {lien.status === 'active' && (
                  <Button
                    variant="danger"
                    onClick={() => handleRelease(lien.id)}
                    disabled={loading}
                  >
                    Release
                  </Button>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className={styles.placeholder}>
            <p className={styles.placeholderTitle}>
              {searchCadastral ? 'No liens found' : 'No liens loaded'}
            </p>
            <p className={styles.placeholderHint}>
              {searchCadastral
                ? 'This parcel has no active or released liens on record'
                : 'Enter a cadastral number above and click Load to view liens'}
            </p>
          </div>
        )}
      </Card>
    </div>
  )
}
