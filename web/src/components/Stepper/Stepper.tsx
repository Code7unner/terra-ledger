import styles from './Stepper.module.css'

interface StepperProps {
  steps: string[]
  currentStep: number
}

export function Stepper({ steps, currentStep }: StepperProps) {
  return (
    <div className={styles.stepper}>
      {steps.map((label, i) => (
        <div key={label} className={styles.stepWrapper}>
          {i > 0 && (
            <div className={`${styles.line} ${i <= currentStep ? styles.lineActive : ''}`} />
          )}
          <div className={`${styles.dot} ${i < currentStep ? styles.dotCompleted : ''} ${i === currentStep ? styles.dotCurrent : ''}`} />
          <span className={`${styles.label} ${i <= currentStep ? styles.labelActive : ''}`}>
            {label}
          </span>
        </div>
      ))}
    </div>
  )
}
