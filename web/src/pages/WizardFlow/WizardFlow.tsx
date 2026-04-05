import { useState, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Stepper } from '../../components/Stepper/Stepper'
import { LandingStep } from './steps/LandingStep'
import { WalletSetupStep } from './steps/WalletSetupStep'
import { RegisterParcelStep } from './steps/RegisterParcelStep'
import { SatelliteStep } from './steps/SatelliteStep'
import { CreditScoreStep } from './steps/CreditScoreStep'
import { SummaryStep } from './steps/SummaryStep'
import { SearchParcelStep } from './steps/SearchParcelStep'
import { RegisterLienStep } from './steps/RegisterLienStep'
import styles from './WizardFlow.module.css'

export type WizardPath = 'none' | 'farmer' | 'lender'

export interface WizardData {
  cadastral: string
  area_ha: number
  land_class: number
  oblast: string
  ndviScore: number | null
  creditScore: number | null
  creditGrade: string | null
  lienRegistered: boolean
}

const FARMER_STEPS = ['Wallet', 'Parcel', 'NDVI', 'Score', 'Profile']
const LENDER_STEPS = ['Search', 'Profile', 'Lien']

const initialData: WizardData = {
  cadastral: '',
  area_ha: 0,
  land_class: 2,
  oblast: '',
  ndviScore: null,
  creditScore: null,
  creditGrade: null,
  lienRegistered: false,
}

export default function WizardFlow() {
  const [searchParams] = useSearchParams()
  const isDemo = searchParams.get('demo') === 'true'

  const [path, setPath] = useState<WizardPath>('none')
  const [step, setStep] = useState(0)
  const [data, setData] = useState<WizardData>(initialData)

  const next = useCallback(() => setStep(s => s + 1), [])
  const back = useCallback(() => setStep(s => Math.max(0, s - 1)), [])
  const updateData = useCallback((partial: Partial<WizardData>) => {
    setData(prev => ({ ...prev, ...partial }))
  }, [])

  const selectPath = useCallback((p: WizardPath) => {
    setPath(p)
    setStep(0)
  }, [])

  const restart = useCallback(() => {
    setPath('none')
    setStep(0)
    setData(initialData)
  }, [])

  if (path === 'none') {
    return (
      <div className={styles.wizard}>
        <LandingStep onSelect={selectPath} />
      </div>
    )
  }

  const steps = path === 'farmer' ? FARMER_STEPS : LENDER_STEPS

  const renderStep = () => {
    if (path === 'farmer') {
      switch (step) {
        case 0: return <WalletSetupStep onNext={next} />
        case 1: return <RegisterParcelStep data={data} isDemo={isDemo} onUpdate={updateData} onNext={next} onBack={back} />
        case 2: return <SatelliteStep cadastral={data.cadastral} isDemo={isDemo} onUpdate={updateData} onNext={next} />
        case 3: return <CreditScoreStep cadastral={data.cadastral} isDemo={isDemo} onNext={next} />
        case 4: return <SummaryStep cadastral={data.cadastral} onRestart={restart} />
        default: return null
      }
    }
    switch (step) {
      case 0: return <SearchParcelStep isDemo={isDemo} onUpdate={updateData} onNext={next} />
      case 1: return <SummaryStep cadastral={data.cadastral} onRestart={restart} showLienButton onRegisterLien={next} />
      case 2: return <RegisterLienStep cadastral={data.cadastral} onBack={back} onDone={restart} />
      default: return null
    }
  }

  return (
    <div className={styles.wizard}>
      <Stepper steps={steps} currentStep={step} />
      <div className={styles.stepContent} key={`${path}-${step}`}>
        {renderStep()}
      </div>
    </div>
  )
}
