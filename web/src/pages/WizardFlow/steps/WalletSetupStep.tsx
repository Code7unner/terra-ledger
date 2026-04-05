import { useEffect } from 'react'
import { useSignAndSend } from '../../../solana/transaction'
import styles from '../WizardFlow.module.css'

interface Props {
  onNext: () => void
}

const QR_URL =
  'https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=https%3A%2F%2Fphantom.app%2Fdownload&bgcolor=141414&color=ededed'

export function WalletSetupStep({ onNext }: Props) {
  const { connected } = useSignAndSend()
  const hasExtension = 'solana' in window

  useEffect(() => {
    if (connected) {
      onNext()
    }
  }, [connected, onNext])

  return (
    <div className={styles.stepCenter}>
      <h2 className={styles.stepTitle}>Connect Your Wallet</h2>
      <p className={styles.stepDesc}>
        Your wallet is like a digital ID for your land. It lets you prove
        ownership and interact with the blockchain securely.
      </p>

      {hasExtension ? (
        <div className={styles.walletHelp}>
          <p className={styles.stepHint}>
            Click <strong>Connect Wallet</strong> in the top right corner to
            continue.
          </p>
        </div>
      ) : (
        <div className={styles.walletHelp}>
          <div className={styles.qrPlaceholder}>
            <img src={QR_URL} alt="Download Phantom wallet" width={200} height={200} />
          </div>
          <p className={styles.stepHint}>
            Scan the QR code or{' '}
            <a
              href="https://phantom.app/download"
              target="_blank"
              rel="noopener noreferrer"
            >
              click here
            </a>{' '}
            to install the Phantom wallet extension.
          </p>
        </div>
      )}
    </div>
  )
}
