import type { HTMLAttributes, ReactNode } from 'react'
import styles from './Card.module.css'

type CardPadding = 'sm' | 'md' | 'lg'

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  padding?: CardPadding
  hoverable?: boolean
  children: ReactNode
}

export function Card({
  padding = 'md',
  hoverable = false,
  children,
  className,
  ...props
}: CardProps) {
  const classes = [
    styles.card,
    styles[`pad-${padding}`],
    hoverable && styles.hoverable,
    className,
  ].filter(Boolean).join(' ')

  return (
    <div className={classes} {...props}>
      {children}
    </div>
  )
}
