import { useState, useRef, useEffect, useCallback } from 'react'
import { useWalletConnection } from '@solana/react-hooks'
import { Button } from '../Button/Button'
import styles from './WalletButton.module.css'

const PHANTOM_URL = 'https://phantom.app/'

type MenuState = 'idle' | 'selectWallet' | 'accountMenu'

function formatAddress(addr: string): string {
  return `${addr.slice(0, 4)}...${addr.slice(-4)}`
}

export function WalletButton() {
  const { wallet, connectors, connect, disconnect, connecting, currentConnector } = useWalletConnection()
  const [menu, setMenu] = useState<MenuState>('idle')
  const [copied, setCopied] = useState(false)
  const wrapperRef = useRef<HTMLDivElement>(null)

  const close = useCallback(() => setMenu('idle'), [])

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        close()
      }
    }
    if (menu !== 'idle') {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [menu, close])

  const handleConnect = async (connectorId: string) => {
    close()
    await connect(connectorId)
  }

  const handleDisconnect = async () => {
    close()
    await disconnect()
  }

  const handleSwitchWallet = async () => {
    await disconnect()
    setMenu('selectWallet')
  }

  const handleCopy = async () => {
    if (!wallet) return
    await navigator.clipboard.writeText(wallet.account.address)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const handleButtonClick = () => {
    if (wallet) {
      setMenu(menu === 'accountMenu' ? 'idle' : 'accountMenu')
    } else {
      setMenu(menu === 'selectWallet' ? 'idle' : 'selectWallet')
    }
  }

  return (
    <div className={styles.wrapper} ref={wrapperRef}>
      {wallet ? (
        <button className={styles.connected} onClick={handleButtonClick} aria-expanded={menu === 'accountMenu'} aria-haspopup="true" aria-label={`Wallet ${formatAddress(wallet.account.address)}`}>
          <span className={styles.dot} aria-hidden="true" />
          <span className={styles.address}>{formatAddress(wallet.account.address)}</span>
          <span className={`${styles.chevron} ${menu === 'accountMenu' ? styles.chevronOpen : ''}`} aria-hidden="true" />
        </button>
      ) : (
        <Button variant="primary" size="sm" loading={connecting} onClick={handleButtonClick}>
          Connect Wallet
        </Button>
      )}

      {menu === 'selectWallet' && (
        <div className={styles.dropdown} role="dialog" aria-label="Select wallet">
          <p className={styles.dropdownTitle}>Select Wallet</p>

          {connectors.length > 0 ? (
            <ul className={styles.connectorList}>
              {connectors.map((c) => (
                <li key={c.id}>
                  <button className={styles.connectorItem} onClick={() => handleConnect(c.id)}>
                    {c.icon && <img src={c.icon} alt={c.name} className={styles.connectorIcon} />}
                    <span className={styles.connectorName}>{c.name}</span>
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <div className={styles.noWallets}>
              <p className={styles.noWalletsText}>No wallets found</p>
              <p className={styles.noWalletsHint}>Install a Solana wallet extension to continue</p>
            </div>
          )}

          <div className={styles.dropdownFooter}>
            <a href={PHANTOM_URL} target="_blank" rel="noopener noreferrer" className={styles.installLink}>
              Get Phantom
            </a>
          </div>
        </div>
      )}

      {menu === 'accountMenu' && wallet && (
        <div className={styles.dropdown} role="menu" aria-label="Wallet actions">
          <div className={styles.accountHeader}>
            <button className={styles.copyRow} onClick={handleCopy}>
              <span className={styles.accountAddress}>{formatAddress(wallet.account.address)}</span>
              <span className={styles.copyIcon}>{copied ? 'Copied!' : 'Copy'}</span>
            </button>
            {currentConnector && (
              <span className={styles.connectorLabel}>via {currentConnector.name}</span>
            )}
          </div>

          <div className={styles.accountActions}>
            <button className={styles.actionItem} onClick={handleSwitchWallet}>
              Switch Wallet
            </button>
            <button className={`${styles.actionItem} ${styles.actionDanger}`} onClick={handleDisconnect}>
              Disconnect
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
