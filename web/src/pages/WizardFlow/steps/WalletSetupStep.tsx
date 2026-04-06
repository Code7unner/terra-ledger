import { useEffect, useRef } from 'react'
import type { WizardData } from '../WizardFlow'
import { useSignAndSend } from '../../../solana/transaction'
import { usePhantomDeeplink } from '../../../solana/usePhantomDeeplink'
import { Button } from '../../../components/Button/Button'
import styles from '../WizardFlow.module.css'

interface Props {
  onUpdate: (partial: Partial<WizardData>) => void
  onNext: () => void
}

const QR_API = 'https://api.qrserver.com/v1/create-qr-code/?size=250x250&bgcolor=141414&color=ededed&data='

export function WalletSetupStep({ onUpdate, onNext }: Props) {
  const { connected, walletAddress: extensionWallet } = useSignAndSend()
  const { connectViaQR, qrUrl, status, walletAddress: deeplinkWallet, error } = usePhantomDeeplink()
  const hasExtension = typeof window !== 'undefined' && 'solana' in window

  // Track whether wallet was already connected when this step mounted.
  // If so, show a "Continue" button instead of auto-advancing (user pressed Back).
  const wasConnectedOnMount = useRef(connected && !!extensionWallet)
  const advancedRef = useRef(false)

  useEffect(() => {
    if (advancedRef.current || wasConnectedOnMount.current) return

    if (connected && extensionWallet) {
      advancedRef.current = true
      onUpdate({ walletAddress: String(extensionWallet) })
      onNext()
      return
    }

    if (status === 'connected' && deeplinkWallet) {
      advancedRef.current = true
      onUpdate({ walletAddress: deeplinkWallet })
      onNext()
    }
  }, [connected, extensionWallet, status, deeplinkWallet, onUpdate, onNext])

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

      {connected && extensionWallet ? (
        <div className={styles.walletHelp}>
          <p className={styles.mintConfirm}>
            Connected: {String(extensionWallet).slice(0, 8)}...{String(extensionWallet).slice(-4)}
          </p>
          <Button onClick={() => { onUpdate({ walletAddress: String(extensionWallet) }); onNext() }}>
            Continue
          </Button>
        </div>
      ) : hasExtension ? (
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

          {status === 'connected' && deeplinkWallet && (
            <p className={styles.mintConfirm}>
              Connected: {deeplinkWallet.slice(0, 8)}...{deeplinkWallet.slice(-4)}
            </p>
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
