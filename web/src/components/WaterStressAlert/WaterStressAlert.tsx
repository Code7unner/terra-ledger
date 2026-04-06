import styles from './WaterStressAlert.module.css'

interface Props {
  waterStress: boolean
}

export function WaterStressAlert({ waterStress }: Props) {
  return (
    <div className={waterStress ? styles.alert : styles.ok}>
      {waterStress ? 'Drought Risk Detected' : 'Adequate Water'}
    </div>
  )
}
