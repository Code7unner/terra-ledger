import { useReducer, useCallback, useEffect } from 'react'
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

interface State {
  path: WizardPath
  step: number
  data: WizardData
}

type Action =
  | { type: 'SELECT_PATH'; path: WizardPath }
  | { type: 'NEXT' }
  | { type: 'BACK' }
  | { type: 'UPDATE_DATA'; partial: Partial<WizardData> }
  | { type: 'UPDATE_AND_NEXT'; partial: Partial<WizardData> }
  | { type: 'GO_TO_STEP'; step: number }
  | { type: 'RESTART' }

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

const initialState: State = {
  path: 'none',
  step: 0,
  data: initialData,
}

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'SELECT_PATH':
      return { ...state, path: action.path, step: 0 }
    case 'NEXT':
      return { ...state, step: state.step + 1 }
    case 'BACK':
      return { ...state, step: Math.max(0, state.step - 1) }
    case 'UPDATE_DATA':
      return { ...state, data: { ...state.data, ...action.partial } }
    case 'UPDATE_AND_NEXT':
      return { ...state, data: { ...state.data, ...action.partial }, step: state.step + 1 }
    case 'GO_TO_STEP':
      return { ...state, step: action.step }
    case 'RESTART':
      return initialState
  }
}

export default function WizardFlow() {
  const [searchParams] = useSearchParams()
  const isDemo = searchParams.get('demo') === 'true'

  const [state, dispatch] = useReducer(reducer, initialState)
  const { path, step, data } = state

  const next = useCallback(() => dispatch({ type: 'NEXT' }), [])
  const back = useCallback(() => dispatch({ type: 'BACK' }), [])
  const updateData = useCallback((partial: Partial<WizardData>) => dispatch({ type: 'UPDATE_DATA', partial }), [])
  const selectPath = useCallback((p: WizardPath) => dispatch({ type: 'SELECT_PATH', path: p }), [])
  const goToStep = useCallback((s: number) => dispatch({ type: 'GO_TO_STEP', step: s }), [])
  const restart = useCallback(() => dispatch({ type: 'RESTART' }), [])

  useEffect(() => {
    const roleParam = searchParams.get('role')
    if (roleParam === 'farmer' || roleParam === 'lender') {
      dispatch({ type: 'SELECT_PATH', path: roleParam })
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  if (path === 'none') {
    return (
      <div className={styles.wizard}>
        <LandingStep onSelect={selectPath} />
      </div>
    )
  }

  const steps = path === 'farmer' ? FARMER_STEPS : LENDER_STEPS
  const cad = data.cadastral

  return (
    <div className={styles.wizard}>
      <Stepper steps={steps} currentStep={step} />
      <div className={styles.stepContent}>
        {path === 'farmer' ? (
          <>
            {step === 0 && <WalletSetupStep onUpdate={updateData} onNext={next} />}
            {step === 1 && <RegisterParcelStep data={data} isDemo={isDemo} onUpdate={updateData} onNext={next} onBack={back} onSkipToSummary={() => goToStep(4)} />}
            {step === 2 && <SatelliteStep cadastral={cad} isDemo={isDemo} onUpdate={updateData} onNext={next} />}
            {step === 3 && <CreditScoreStep cadastral={cad} isDemo={isDemo} onNext={next} />}
            {step === 4 && <SummaryStep cadastral={cad} onRestart={restart} />}
          </>
        ) : (
          <>
            {step === 0 && <SearchParcelStep isDemo={isDemo} onUpdate={updateData} onNext={next} />}
            {step === 1 && <SummaryStep cadastral={cad} onRestart={restart} showLienButton onRegisterLien={next} />}
            {step === 2 && <RegisterLienStep cadastral={cad} onBack={back} onDone={restart} />}
          </>
        )}
      </div>
    </div>
  )
}
