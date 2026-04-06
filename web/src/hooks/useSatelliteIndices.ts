import { useState, useCallback } from 'react'

const API = import.meta.env.VITE_API_URL || 'http://localhost:3000'
const KEY = import.meta.env.VITE_API_KEY || ''

export interface AHI {
  composite: number
  ndvi: number
  ndwi_norm: number
  evi_norm: number
  lai_norm: number
  water_stress: boolean
}

export function useSatelliteIndices() {
  const [ahi, setAhi] = useState<AHI | null>(null)
  const [loading, setLoading] = useState(false)

  const fetchIndices = useCallback(async (cadastral: string) => {
    setLoading(true)
    try {
      const headers: Record<string, string> = {}
      if (KEY) headers['X-API-Key'] = KEY
      const res = await fetch(`${API}/api/v1/parcels/${cadastral}/indices`, {
        headers,
      })
      if (res.ok) {
        const body = await res.json()
        setAhi(body.ahi ?? null)
      }
    } finally {
      setLoading(false)
    }
  }, [])

  return { ahi, loading, fetchIndices }
}
