import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { MapContainer, TileLayer, CircleMarker, Popup, useMap } from 'react-leaflet'
import { useMapParcels, type MapParcel } from '../../../hooks/useMapParcels'
import type { WizardPath } from '../WizardFlow'
import 'leaflet/dist/leaflet.css'
import styles from '../WizardFlow.module.css'

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

function InvalidateSize() {
  const map = useMap()
  useEffect(() => {
    const timer = setTimeout(() => map.invalidateSize(), 100)
    return () => clearTimeout(timer)
  }, [map])
  return null
}

function ParcelMarker({ parcel }: { parcel: MapParcel }) {
  if (parcel.latitude == null || parcel.longitude == null) return null
  const color = colorForClass(parcel.land_class)
  const radius = radiusForArea(parcel.area_ha)

  return (
    <CircleMarker
      center={[parcel.latitude, parcel.longitude]}
      radius={radius}
      pathOptions={{ color, fillColor: color, fillOpacity: 0.35, weight: 2 }}
    >
      <Popup>
        <div className={styles.mapPopup}>
          <strong>{parcel.cadastral_number}</strong>
          Area: {parcel.area_ha} ha<br />
          Land class: {parcel.land_class}<br />
          {parcel.oblast && <>Oblast: {parcel.oblast}<br /></>}
          <Link className={styles.mapPopupLink} to={`/app/parcel/${parcel.cadastral_number}`}>
            View details
          </Link>
        </div>
      </Popup>
    </CircleMarker>
  )
}

interface Props {
  onSelect: (path: WizardPath) => void
}

export function LandingStep({ onSelect }: Props) {
  const { parcels, loading, fetchParcels } = useMapParcels()

  useEffect(() => {
    fetchParcels()
  }, [fetchParcels])

  return (
    <div className={styles.landing}>
      <div className={styles.hero}>
        <h1 className={styles.heroTitle}>
          <span className={styles.heroSlash}>//</span> Agricultural Credit Intelligence
        </h1>
        <p className={styles.heroSub}>
          Satellite-verified productivity meets on-chain credit scoring for Kazakhstan
        </p>
      </div>
      <div className={styles.roleCards}>
        <button className={styles.roleCard} onClick={() => onSelect('farmer')}>
          <h2 className={styles.roleTitle}>I'm a Farmer</h2>
          <p className={styles.roleDesc}>Register your land and get a credit score based on satellite data</p>
        </button>
        <button className={styles.roleCard} onClick={() => onSelect('lender')}>
          <h2 className={styles.roleTitle}>I'm a Lender</h2>
          <p className={styles.roleDesc}>Search parcels and assess credit risk instantly</p>
        </button>
      </div>

      <div className={styles.mapSection}>
        <h2 className={styles.mapHeading}>Browse Registered Parcels</h2>
        {loading && <div className={styles.mapLoading}>Loading parcels...</div>}
        <div className={styles.mapWrap}>
          <MapContainer
            center={KZ_CENTER}
            zoom={DEFAULT_ZOOM}
            className={styles.map}
            scrollWheelZoom
          >
            <InvalidateSize />
            <TileLayer
              attribution='&copy; <a href="https://carto.com/">CARTO</a>'
              url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
            />
            {parcels.map((p) => (
              <ParcelMarker key={p.id} parcel={p} />
            ))}
          </MapContainer>
          <div className={styles.mapLegend}>
            <div className={styles.mapLegendTitle}>Land Class</div>
            <div className={styles.mapLegendItem}>
              <span className={styles.mapLegendDot} style={{ background: '#22c55e' }} />
              Class 1-2 (high)
            </div>
            <div className={styles.mapLegendItem}>
              <span className={styles.mapLegendDot} style={{ background: '#eab308' }} />
              Class 3-4 (medium)
            </div>
            <div className={styles.mapLegendItem}>
              <span className={styles.mapLegendDot} style={{ background: '#ef4444' }} />
              Class 5+ (low)
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
