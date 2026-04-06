import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import leafIcon from '../../../favicon.svg'
import styles from './Landing.module.css'

const GITHUB_URL = 'https://github.com/code7unner/terra-ledger'
const TERRA_TOKEN_EXPLORER =
  'https://explorer.solana.com/address/2eAqpJ7yjso7FDA4sDQLJQioNCRuoYSUeha2Y88NRRMX?cluster=devnet'
const LIEN_REGISTRY_EXPLORER =
  'https://explorer.solana.com/address/3qYHSTPeRLRDfWmtzEhiaHpT2kchgW8GqaYcwmDbKnq4?cluster=devnet'

function useCountUp(target: number, duration: number, decimals: number, triggerRef: React.RefObject<HTMLElement | null>) {
  const [value, setValue] = useState(0)
  const triggered = useRef(false)

  useEffect(() => {
    const el = triggerRef.current
    if (!el) return

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && !triggered.current) {
          triggered.current = true
          const start = performance.now()
          const step = (now: number) => {
            const t = Math.min((now - start) / duration, 1)
            const ease = 1 - Math.pow(1 - t, 3)
            setValue(Math.round(target * ease * Math.pow(10, decimals)) / Math.pow(10, decimals))
            if (t < 1) requestAnimationFrame(step)
          }
          requestAnimationFrame(step)
        }
      },
      { threshold: 0.1 },
    )
    observer.observe(el)
    return () => observer.disconnect()
  }, [target, duration, decimals, triggerRef])

  return value
}

export default function Landing() {
  const [scrolled, setScrolled] = useState(false)
  const statsRef = useRef<HTMLElement>(null)

  const creditGap = useCountUp(2.7, 1500, 1, statsRef)
  const rejection = useCountUp(67, 1500, 0, statsRef)
  const verifyDays = useCountUp(21, 1500, 0, statsRef)

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 80)
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  useEffect(() => {
    const els = document.querySelectorAll('[data-animate]')
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add(styles.visible)
          }
        })
      },
      { threshold: 0.1 },
    )
    els.forEach((el) => observer.observe(el))
    return () => observer.disconnect()
  }, [])

  return (
    <div className={styles.page}>
      {/* Header */}
      <header className={styles.header}>
        <Link to="/" className={styles.logo}>
          <img src={leafIcon} className={styles.logoIcon} alt="" aria-hidden="true" />
          TerraLedger
        </Link>
        <Link to="/app" className={styles.headerCta}>
          Launch App
        </Link>
      </header>

      {/* Hero */}
      <section className={styles.hero}>
        <div className={styles.heroBackground} />
        <div className={styles.heroContent}>
          <span className={styles.eyebrow}>Built on Solana / Devnet</span>
          <h1 className={styles.headline}>
            <span className={styles.headlineAccent}>//</span> Agricultural Credit
            Intelligence
          </h1>
          <p className={styles.subtitle}>
            Satellite-verified productivity certificates, AI credit scoring, and on-chain
            lien registry for Kazakhstan's 18 million hectares of underserved farmland.
          </p>
          <div className={styles.heroCtas}>
            <Link to="/app" className={styles.btnPrimary}>
              Launch App
            </Link>
            <a
              href={GITHUB_URL}
              target="_blank"
              rel="noopener noreferrer"
              className={styles.btnSecondary}
            >
              GitHub
            </a>
          </div>
        </div>
        <div
          className={styles.scrollIndicator}
          style={{ opacity: scrolled ? 0 : undefined }}
        >
          <div className={styles.scrollMouse} />
          <span className={styles.scrollLabel}>scroll</span>
        </div>
      </section>

      {/* Stats */}
      <section className={styles.stats} ref={statsRef} data-animate>
        <div className={styles.statsInner}>
          <div className={styles.statCard}>
            <div className={styles.statValue}>${creditGap.toFixed(1)}B</div>
            <div className={styles.statLabel}>Agricultural Credit Gap</div>
          </div>
          <div className={styles.statCard}>
            <div className={styles.statValue}>{Math.round(rejection)}%</div>
            <div className={styles.statLabel}>Loan Rejection Rate</div>
          </div>
          <div className={styles.statCard}>
            <div className={styles.statValue}>{Math.round(verifyDays)}-Day</div>
            <div className={styles.statLabel}>Manual Verification</div>
          </div>
        </div>
      </section>

      {/* Pipeline */}
      <section className={styles.pipeline} data-animate>
        <div className={styles.pipelineInner}>
          <h2 className={styles.sectionHeading}>How It Works</h2>
          <div className={styles.pipelineGrid}>
            <div className={styles.featureCard}>
              <span className={styles.stepBadge}>01</span>
              <h3 className={styles.featureTitle}>Satellite Verification</h3>
              <p className={styles.featureDesc}>
                Real Sentinel-2 data from Copernicus. NDVI, NDWI, EVI indices with cloud
                masking over a 12-month time series per parcel.
              </p>
            </div>
            <div className={styles.featureCard}>
              <span className={styles.stepBadge}>02</span>
              <h3 className={styles.featureTitle}>AI Credit Scoring</h3>
              <p className={styles.featureDesc}>
                Claude API analyzes satellite indices and land metadata. Outputs a 0-100
                score, collateral grade, recommended LTV, and risk factors.
              </p>
            </div>
            <div className={styles.featureCard}>
              <span className={styles.stepBadge}>03</span>
              <h3 className={styles.featureTitle}>On-Chain Registry</h3>
              <p className={styles.featureDesc}>
                AI score written to Solana PDA. Non-transferable NDVI certificates via
                Token-2022. Atomic double-pledge prevention through CPI.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Roles */}
      <section className={styles.roles} data-animate>
        <div className={styles.rolesInner}>
          <h2 className={styles.sectionHeading}>Get Started</h2>
          <div className={styles.roleGrid}>
            <Link to="/app?role=farmer&demo=true" className={styles.roleCard}>
              <h3 className={styles.roleTitle}>I'm a Farmer</h3>
              <p className={styles.roleDesc}>
                Register your land parcel and get a satellite-verified credit score in
                minutes
              </p>
              <span className={styles.roleArrow}>&rarr;</span>
            </Link>
            <Link to="/app?role=lender&demo=true" className={styles.roleCard}>
              <h3 className={styles.roleTitle}>I'm a Lender</h3>
              <p className={styles.roleDesc}>
                Search verified parcels and assess credit risk with on-chain transparency
              </p>
              <span className={styles.roleArrow}>&rarr;</span>
            </Link>
          </div>
        </div>
      </section>

      {/* Links */}
      <section className={styles.links} data-animate>
        <div className={styles.linksInner}>
          <a
            href={GITHUB_URL}
            target="_blank"
            rel="noopener noreferrer"
            className={styles.linkPill}
          >
            GitHub
          </a>
          <a
            href={TERRA_TOKEN_EXPLORER}
            target="_blank"
            rel="noopener noreferrer"
            className={styles.linkPill}
          >
            terra_token
          </a>
          <a
            href={LIEN_REGISTRY_EXPLORER}
            target="_blank"
            rel="noopener noreferrer"
            className={styles.linkPill}
          >
            lien_registry
          </a>
        </div>
      </section>

      {/* Footer */}
      <footer className={styles.footer}>
        Built for Decentrathon 5 &mdash; National Solana Hackathon Kazakhstan
      </footer>
    </div>
  )
}
