import { useState, useCallback } from 'react'
import { get } from '../api/client'

export interface NDVICertificate {
  id: string
  parcel_id: string
  cadastral_number: string
  season: string
  ndvi_score: number
  crop_type?: string
  yield_t_ha?: number
  sentinel_scene_id?: string
  on_chain_address?: string
  tx_signature?: string
  minted_at: string
}

export interface Encumbrance {
  id: string
  parcel_id: string
  cadastral_number: string
  lender_wallet: string
  lender_name?: string
  amount_tenge: number
  notary_cert_hash?: string
  on_chain_address?: string
  tx_signature?: string
  status: 'active' | 'released' | 'disputed'
  registered_at: string
  released_at?: string
}

export interface CreditProfile {
  parcel: {
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
  productivity: {
    certificates: NDVICertificate[]
    ndvi_trend: string
    dormancy_risk: string
  }
  encumbrances: {
    active_liens: Encumbrance[]
    lien_count_historical: number
    double_pledge_risk: boolean
  }
  credit_intelligence?: {
    id: string
    ai_score: number
    recommended_ltv: number
    collateral_grade: string
    estimated_value_tenge: number
    model_version: string
    explanation: string
    risk_factors: string[]
    computed_at: string
  }
}

export function useCreditProfile() {
  const [data, setData] = useState<CreditProfile | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchProfile = useCallback(async (cadastral: string) => {
    setLoading(true)
    setError(null)
    try {
      const resp = await get<CreditProfile>(`/api/v1/parcels/${encodeURIComponent(cadastral)}/profile`)
      setData(resp)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch credit profile')
    } finally {
      setLoading(false)
    }
  }, [])

  return { data, loading, error, fetchProfile }
}
