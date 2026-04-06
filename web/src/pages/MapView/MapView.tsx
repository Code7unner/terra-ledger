import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { MapContainer, TileLayer, CircleMarker, Popup, useMap } from 'react-leaflet'
import { useMapParcels, type MapParcel } from '../../hooks/useMapParcels'
import 'leaflet/dist/leaflet.css'
import styles from './MapView.module.css'

const KZ_CENTER: [number, number] = [48.0, 68.0]
const DEFAULT_ZOOM = 5

function colorForClass(landClass: number): string {
  if (landClass <= 2) return '#22c55e'
  if (landClass <= 4) return '#eab308'
  return '#ef4444'
}

function radiusForArea(area: number): number {
  const base = Math.sqrt(area) * 2
  return Math.max(6, Math.min(base, 30))
}

function ParcelMarker({ parcel }: { parcel: MapParcel }) {
  if (parcel.latitude == null || parcel.longitude == null) return null

  const color = colorForClass(parcel.land_class)
  const radius = radiusForArea(parcel.area_ha)

  return (
    <CircleMarker
      center={[parcel.latitude, parcel.longitude]}
      radius={radius}
      pathOptions={{
        color,
        fillColor: color,
        fillOpacity: 0.35,
        weight: 2,
      }}
    >
      <Popup>
        <div className={styles.popup}>
          <strong>{parcel.cadastral_number}</strong>
          Area: {parcel.area_ha} ha<br />
          Land class: {parcel.land_class}<br />
          {parcel.oblast && <>Oblast: {parcel.oblast}<br /></>}
          <Link
            className={styles.popupLink}
            to={`/app/parcel/${parcel.cadastral_number}`}
          >
            View details
          </Link>
        </div>
      </Popup>
    </CircleMarker>
  )
}

function InvalidateSize() {
  const map = useMap()
  useEffect(() => {
    const timer = setTimeout(() => map.invalidateSize(), 100)
    return () => clearTimeout(timer)
  }, [map])
  return null
}

function Legend() {
  return (
    <div className={styles.legend}>
      <div className={styles.legendTitle}>Land Class</div>
      <div className={styles.legendItem}>
        <span className={styles.legendDot} style={{ background: '#22c55e' }} />
        Class 1-2 (high)
      </div>
      <div className={styles.legendItem}>
        <span className={styles.legendDot} style={{ background: '#eab308' }} />
        Class 3-4 (medium)
      </div>
      <div className={styles.legendItem}>
        <span className={styles.legendDot} style={{ background: '#ef4444' }} />
        Class 5+ (low)
      </div>
    </div>
  )
}

export default function MapView() {
  const { parcels, loading, fetchParcels } = useMapParcels()

  useEffect(() => {
    fetchParcels()
  }, [fetchParcels])

  return (
    <div className={styles.container}>
      <h1 className={styles.title}>Parcel Map</h1>

      {loading && <div className={styles.loading}>Loading parcels...</div>}

      <div className={styles.mapWrap}>
        <MapContainer
          center={KZ_CENTER}
          zoom={DEFAULT_ZOOM}
          className={styles.map}
          scrollWheelZoom
        >
          <InvalidateSize />
          <TileLayer
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          />
          {parcels.map((p) => (
            <ParcelMarker key={p.id} parcel={p} />
          ))}
        </MapContainer>
        <Legend />
      </div>
    </div>
  )
}
