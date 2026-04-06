import { useState, useCallback } from 'react'
import { get } from '../api/client'

export interface AgentDecision {
  id: string
  cadastral_number: string
  previous_score: number | null
  new_score: number
  previous_grade: string | null
  new_grade: string
  reason: string
  tx_signature: string
  decided_at: string
}

interface AgentDecisionsResponse {
  decisions: AgentDecision[]
}

export function useAgentDecisions() {
  const [decisions, setDecisions] = useState<AgentDecision[]>([])
  const [loading, setLoading] = useState(false)

  const fetchRecent = useCallback(async () => {
    setLoading(true)
    try {
      const body = await get<AgentDecisionsResponse>('/api/v1/agent/decisions')
      setDecisions(body.decisions ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchByParcel = useCallback(async (cadastral: string) => {
    setLoading(true)
    try {
      const body = await get<AgentDecisionsResponse>(
        `/api/v1/agent/decisions/${encodeURIComponent(cadastral)}`,
      )
      setDecisions(body.decisions ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  return { decisions, loading, fetchRecent, fetchByParcel }
}
