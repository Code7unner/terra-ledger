import { useState } from 'react'
import type { WizardData } from '../WizardFlow'
import { get } from '../../../api/client'
import { useToast } from '../../../components/Toast/useToast'
import { Input } from '../../../components/Input/Input'
import { Button } from '../../../components/Button/Button'
import styles from '../WizardFlow.module.css'

interface Props {
  isDemo: boolean
  onUpdate: (partial: Partial<WizardData>) => void
  onNext: () => void
}

const DEMO_PARCELS = ['KZ11-0033-001', 'KZ11-0033-002', 'KZ11-0033-003']

export function SearchParcelStep({ onUpdate, onNext }: Props) {
  const [query, setQuery] = useState('')
  const [searching, setSearching] = useState(false)
  const { toast } = useToast()

  const search = async (cadastral: string) => {
    setSearching(true)
    try {
      await get(`/api/v1/parcels/${encodeURIComponent(cadastral)}`)
      onUpdate({ cadastral })
      onNext()
    } catch {
      toast(`Parcel ${cadastral} not found`)
    } finally {
      setSearching(false)
    }
  }

  const handleSearch = () => {
    const trimmed = query.trim()
    if (!trimmed) return
    search(trimmed)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleSearch()
  }

  return (
    <div className={styles.stepCenter}>
      <h2 className={styles.stepTitle}>Search Parcel</h2>
      <p className={styles.stepDesc}>
        Enter a cadastral number to look up a land parcel and view its credit profile.
      </p>

      <div className={styles.stepForm}>
        <Input
          label="Cadastral Number"
          placeholder="e.g. KZ11-0033-001"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
        />
        <Button onClick={handleSearch} disabled={!query.trim() || searching} loading={searching}>
          Search
        </Button>
      </div>

      <div>
        <p className={styles.stepHint}>Or try a demo parcel:</p>
        <div className={styles.demoChips}>
          {DEMO_PARCELS.map((id) => (
            <button
              key={id}
              className={styles.chip}
              onClick={() => search(id)}
              disabled={searching}
            >
              {id}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
