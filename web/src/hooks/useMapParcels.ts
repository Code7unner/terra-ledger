import { useState, useCallback } from 'react'

const API = import.meta.env.VITE_API_URL || 'http://localhost:3000'
const KEY = import.meta.env.VITE_API_KEY || ''

export interface MapParcel {
  id: string
  cadastral_number: string
  area_ha: number
  land_class: number
  oblast: string
  latitude: number | null
  longitude: number | null
}

export function useMapParcels() {
  const [parcels, setParcels] = useState<MapParcel[]>([])
  const [loading, setLoading] = useState(false)

  const fetchParcels = useCallback(async () => {
    setLoading(true)
    try {
      const headers: Record<string, string> = {}
      if (KEY) headers['X-API-Key'] = KEY
      const res = await globalThis.fetch(`${API}/api/v1/parcels`, { headers })
      if (res.ok) {
        const body = await res.json()
        setParcels(body.parcels ?? [])
      }
    } finally {
      setLoading(false)
    }
  }, [])

  return { parcels, loading, fetchParcels }
}
