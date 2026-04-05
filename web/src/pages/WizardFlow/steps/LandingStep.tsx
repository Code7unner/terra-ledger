import type { WizardPath } from '../WizardFlow'
import styles from '../WizardFlow.module.css'

interface Props {
  onSelect: (path: WizardPath) => void
}

export function LandingStep({ onSelect }: Props) {
  return (
    <div className={styles.landing}>
      <div className={styles.hero}>
        <h1 className={styles.heroTitle}>
          <span className={styles.heroSlash}>//</span> Agricultural Credit Intelligence
        </h1>
        <p className={styles.heroSub}>
          Satellite-verified productivity meets on-chain credit scoring for Kazakhstan
        </p>
      </div>
      <div className={styles.roleCards}>
        <button className={styles.roleCard} onClick={() => onSelect('farmer')}>
          <h2 className={styles.roleTitle}>I'm a Farmer</h2>
          <p className={styles.roleDesc}>Register your land and get a credit score based on satellite data</p>
        </button>
        <button className={styles.roleCard} onClick={() => onSelect('lender')}>
          <h2 className={styles.roleTitle}>I'm a Lender</h2>
          <p className={styles.roleDesc}>Search parcels and assess credit risk instantly</p>
        </button>
      </div>
    </div>
  )
}
