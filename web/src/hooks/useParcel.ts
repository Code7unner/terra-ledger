import { useState, useCallback } from 'react'
import { get, post } from '../api/client'

export interface Parcel {
  id: string
  cadastral_number: string
  owner_wallet: string
  on_chain_address?: string
  area_ha: number
  land_class: number
  kyc_verified: boolean
  oblast?: string
  rayon?: string
  holder_name?: string
  registered_at: string
  updated_at: string
}

export interface RegisterParcelInput {
  cadastral_number: string
  owner_wallet: string
  area_ha: number
  land_class: number
  oblast: string
  rayon: string
  holder_name: string
  holder_iin_hash: string
}

export function useParcel() {
  const [data, setData] = useState<Parcel | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchParcel = useCallback(async (cadastral: string) => {
    setLoading(true)
    setError(null)
    try {
      const resp = await get<Parcel>(`/api/v1/parcels/${encodeURIComponent(cadastral)}`)
      setData(resp)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch parcel')
    } finally {
      setLoading(false)
    }
  }, [])

  const registerParcel = useCallback(async (input: RegisterParcelInput) => {
    setLoading(true)
    setError(null)
    try {
      const resp = await post<Parcel>('/api/v1/parcels', input)
      setData(resp)
      return resp
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to register parcel')
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  return { data, loading, error, fetchParcel, registerParcel }
}
