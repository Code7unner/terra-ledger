import { useState, useCallback } from 'react'
import { get, post } from '../api/client'
import type { Encumbrance } from './useCreditProfile'

export type { Encumbrance as Lien }

export function useLien() {
  const [liens, setLiens] = useState<Encumbrance[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchLiens = useCallback(async (cadastral: string) => {
    setLoading(true)
    setError(null)
    try {
      const resp = await get<Encumbrance[]>(`/api/v1/parcels/${encodeURIComponent(cadastral)}/liens`)
      setLiens(resp)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch liens')
    } finally {
      setLoading(false)
    }
  }, [])

  const registerLien = useCallback(async (payload: {
    cadastral_number: string
    lender_wallet: string
    amount_tenge: number
    notary_cert_hash: string
  }) => {
    setLoading(true)
    setError(null)
    try {
      const resp = await post<Encumbrance>('/api/v1/liens', payload)
      setLiens((prev) => [...prev, resp])
      return resp
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to register lien')
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  const releaseLien = useCallback(async (lienId: string) => {
    setLoading(true)
    setError(null)
    try {
      await post(`/api/v1/liens/${encodeURIComponent(lienId)}/release`, {})
      setLiens((prev) => prev.map((l) =>
        l.id === lienId ? { ...l, status: 'released' as const } : l
      ))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to release lien')
    } finally {
      setLoading(false)
    }
  }, [])

  return { liens, loading, error, fetchLiens, registerLien, releaseLien }
}
