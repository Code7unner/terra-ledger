import { NavLink } from 'react-router-dom'
import { WalletButton } from '../WalletButton/WalletButton'
import leafIcon from '../../../favicon.svg'
import styles from './TopBar.module.css'

export function TopBar() {
  return (
    <header className={styles.topbar}>
      <NavLink to="/" className={styles.logo}>
        <img src={leafIcon} className={styles.logoIcon} alt="" aria-hidden="true" />
        TerraLedger
      </NavLink>
      <div className={styles.right}>
        <WalletButton />
      </div>
    </header>
  )
}
