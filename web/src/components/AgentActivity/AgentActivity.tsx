import { useEffect } from 'react'
import { useAgentDecisions } from '../../hooks/useAgentDecisions'
import { Card } from '../Card/Card'
import styles from './AgentActivity.module.css'

interface AgentActivityProps {
  cadastral?: string
}

export function AgentActivity({ cadastral }: AgentActivityProps) {
  const { decisions, loading, fetchRecent, fetchByParcel } = useAgentDecisions()

  useEffect(() => {
    if (cadastral) {
      fetchByParcel(cadastral)
    } else {
      fetchRecent()
    }
  }, [cadastral, fetchByParcel, fetchRecent])

  if (loading || decisions.length === 0) {
    return null
  }

  return (
    <Card padding="lg">
      <div className={styles.wrapper}>
        <h2 className={styles.title}>AI Agent Activity</h2>
        <div className={styles.timeline}>
          {decisions.map((d) => (
            <div key={d.id} className={styles.entry}>
              <div className={styles.entryHeader}>
                <span className={styles.grade}>{d.new_grade}</span>
                <span className={styles.scoreChange}>
                  {d.previous_score != null ? (
                    <>
                      {d.previous_score}
                      <span className={styles.arrow}>&rarr;</span>
                      {d.new_score}
                    </>
                  ) : (
                    d.new_score
                  )}
                </span>
                <span className={styles.timestamp}>
                  {new Date(d.decided_at).toLocaleDateString()}
                </span>
              </div>
              {d.reason && <p className={styles.reason}>{d.reason}</p>}
              {d.tx_signature && (
                <a
                  className={styles.txLink}
                  href={`https://explorer.solana.com/tx/${d.tx_signature}?cluster=devnet`}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {d.tx_signature.slice(0, 16)}...
                </a>
              )}
            </div>
          ))}
        </div>
      </div>
    </Card>
  )
}
