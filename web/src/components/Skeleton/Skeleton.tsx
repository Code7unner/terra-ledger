import styles from './Skeleton.module.css'

interface SkeletonProps {
  width?: string
  height?: string
}

export function Skeleton({ width = '100%', height = '20px' }: SkeletonProps) {
  return <div className={styles.skeleton} style={{ width, height }} />
}
