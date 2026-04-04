import type { InputHTMLAttributes } from 'react'
import styles from './Input.module.css'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  suffix?: string
  error?: string
}

export function Input({ label, suffix, error, className, ...props }: InputProps) {
  return (
    <div className={`${styles.wrapper} ${className ?? ''}`}>
      {label && <label className={styles.label}>{label}</label>}
      <div className={`${styles.field} ${error ? styles.fieldError : ''}`}>
        <input className={styles.input} {...props} />
        {suffix && <span className={styles.suffix}>{suffix}</span>}
      </div>
      {error && <span className={styles.error}>{error}</span>}
    </div>
  )
}
