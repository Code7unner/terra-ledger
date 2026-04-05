import { NavLink } from 'react-router-dom'
import { WalletButton } from '../WalletButton/WalletButton'
import styles from './TopBar.module.css'

export function TopBar() {
  return (
    <header className={styles.topbar}>
      <NavLink to="/" className={styles.logo}>TerraLedger</NavLink>
      <div className={styles.right}>
        <WalletButton />
      </div>
    </header>
  )
}
