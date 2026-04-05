import { useId, type InputHTMLAttributes } from 'react'
import styles from './Input.module.css'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  suffix?: string
  error?: string
}

export function Input({ label, suffix, error, className, id, ...props }: InputProps) {
  const autoId = useId()
  const inputId = id ?? autoId

  return (
    <div className={`${styles.wrapper} ${className ?? ''}`}>
      {label && <label htmlFor={inputId} className={styles.label}>{label}</label>}
      <div className={`${styles.field} ${error ? styles.fieldError : ''}`}>
        <input id={inputId} className={styles.input} aria-invalid={!!error} aria-describedby={error ? `${inputId}-error` : undefined} {...props} />
        {suffix && <span className={styles.suffix}>{suffix}</span>}
      </div>
      {error && <span id={`${inputId}-error`} className={styles.error} role="alert">{error}</span>}
    </div>
  )
}
