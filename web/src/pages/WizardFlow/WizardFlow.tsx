import { useState, useCallback, useRef } from 'react'
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
  walletAddress: string | null
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
  walletAddress: null,
  lienRegistered: false,
}

export default function WizardFlow() {
  const [searchParams] = useSearchParams()
  const isDemo = searchParams.get('demo') === 'true'

  const [path, setPath] = useState<WizardPath>('none')
  const [step, setStep] = useState(0)
  const [data, setData] = useState<WizardData>(initialData)
  const dataRef = useRef<WizardData>(initialData)

  const next = useCallback(() => setStep(s => s + 1), [])
  const back = useCallback(() => setStep(s => Math.max(0, s - 1)), [])
  const updateData = useCallback((partial: Partial<WizardData>) => {
    // Update ref immediately (synchronous) so downstream renders see it
    dataRef.current = { ...dataRef.current, ...partial }
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
    dataRef.current = initialData
  }, [])

  if (path === 'none') {
    return (
      <div className={styles.wizard}>
        <LandingStep onSelect={selectPath} />
      </div>
    )
  }

  const steps = path === 'farmer' ? FARMER_STEPS : LENDER_STEPS

  const getCadastral = () => dataRef.current.cadastral

  return (
    <div className={styles.wizard}>
      <Stepper steps={steps} currentStep={step} />
      <div className={styles.stepContent}>
        {path === 'farmer' ? (
          <>
            {step === 0 && <WalletSetupStep onUpdate={updateData} onNext={next} />}
            {step === 1 && <RegisterParcelStep data={data} isDemo={isDemo} onUpdate={updateData} onNext={next} onBack={back} />}
            {step === 2 && <SatelliteStep cadastral={getCadastral()} isDemo={isDemo} onUpdate={updateData} onNext={next} />}
            {step === 3 && <CreditScoreStep cadastral={getCadastral()} isDemo={isDemo} onNext={next} />}
            {step === 4 && <SummaryStep cadastral={getCadastral()} onRestart={restart} />}
          </>
        ) : (
          <>
            {step === 0 && <SearchParcelStep isDemo={isDemo} onUpdate={updateData} onNext={next} />}
            {step === 1 && <SummaryStep cadastral={getCadastral()} onRestart={restart} showLienButton onRegisterLien={next} />}
            {step === 2 && <RegisterLienStep cadastral={getCadastral()} onBack={back} onDone={restart} />}
          </>
        )}
      </div>
    </div>
  )
}
