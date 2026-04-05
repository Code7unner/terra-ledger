import { NavLink } from 'react-router-dom'
import { WalletButton } from '../WalletButton/WalletButton'
import styles from './TopBar.module.css'

const navItems = [
  { to: '/', label: 'Lender' },
  { to: '/liens', label: 'Liens' },
  { to: '/farmer', label: 'Farmer' },
  { to: '/farmer/consent', label: 'Consent' },
]

export function TopBar() {
  return (
    <header className={styles.topbar}>
      <NavLink to="/" className={styles.logo}>TerraLedger</NavLink>
      <nav className={styles.nav} aria-label="Main navigation">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end
            className={({ isActive }) =>
              `${styles.navLink} ${isActive ? styles.navLinkActive : ''}`
            }
          >
            {item.label}
          </NavLink>
        ))}
      </nav>
      <div className={styles.right}>
        <WalletButton />
      </div>
    </header>
  )
}
