import { useEffect, useRef } from 'react'
import { useSignAndSend } from '../../../solana/transaction'
import { usePhantomDeeplink } from '../../../solana/usePhantomDeeplink'
import { Button } from '../../../components/Button/Button'
import styles from '../WizardFlow.module.css'

interface Props {
  onNext: () => void
}

const QR_API = 'https://api.qrserver.com/v1/create-qr-code/?size=250x250&bgcolor=141414&color=ededed&data='

export function WalletSetupStep({ onNext }: Props) {
  const { connected } = useSignAndSend()
  const { connectViaQR, qrUrl, status, walletAddress, error } = usePhantomDeeplink()
  const hasExtension = typeof window !== 'undefined' && 'solana' in window
  const advancedRef = useRef(false)

  // Auto-advance when wallet connects (via extension or deeplink)
  useEffect(() => {
    if ((connected || status === 'connected') && !advancedRef.current) {
      advancedRef.current = true
      onNext()
    }
  }, [connected, status, onNext])

  // Auto-start QR flow if no extension
  useEffect(() => {
    if (!hasExtension && status === 'idle') {
      connectViaQR()
    }
  }, [hasExtension, status, connectViaQR])

  return (
    <div className={styles.stepCenter}>
      <h2 className={styles.stepTitle}>
        <span className={styles.heroSlash}>//</span> Connect Your Wallet
      </h2>
      <p className={styles.stepDesc}>
        Your wallet is like a digital ID for your land. It proves ownership without paperwork.
      </p>

      {hasExtension ? (
        <div className={styles.walletHelp}>
          <p className={styles.stepHint}>
            Click <strong>Connect Wallet</strong> in the top right corner to continue.
          </p>
        </div>
      ) : (
        <div className={styles.walletHelp}>
          {status === 'waiting' && qrUrl && (
            <>
              <div className={styles.qrPlaceholder}>
                <img
                  src={`${QR_API}${encodeURIComponent(qrUrl)}`}
                  alt="Connect with Phantom"
                  width={250}
                  height={250}
                />
              </div>
              <p className={styles.stepHint}>
                Open <strong>Phantom</strong> on your phone and scan this QR code
              </p>
              <div className={styles.loadingAnim}>
                <div className={styles.spinner} />
                <p className={styles.loadingMsg}>Waiting for approval...</p>
              </div>
            </>
          )}

          {status === 'connected' && walletAddress && (
            <div>
              <p className={styles.mintConfirm}>
                Connected: {walletAddress.slice(0, 8)}...{walletAddress.slice(-4)}
              </p>
            </div>
          )}

          {status === 'error' && (
            <div className={styles.walletHelp}>
              <p className={styles.error}>{error}</p>
              <Button variant="secondary" onClick={connectViaQR}>Try Again</Button>
            </div>
          )}

          {status === 'idle' && (
            <div className={styles.walletHelp}>
              <div className={styles.spinner} />
              <p className={styles.stepHint}>Preparing connection...</p>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
