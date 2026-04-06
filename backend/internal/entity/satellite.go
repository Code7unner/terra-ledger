package entity

import "time"

// SatelliteIndices holds multi-index observation data from a single time period.
type SatelliteIndices struct {
	NDVI         float64   `json:"ndvi"`
	NDWI         float64   `json:"ndwi"`
	EVI          float64   `json:"evi"`
	LAI          float64   `json:"lai"`
	CloudFreePct float64   `json:"cloud_free_pct"`
	SampleCount  int       `json:"sample_count"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
}

// SatelliteTimeSeries holds a sequence of multi-index observations for a parcel.
type SatelliteTimeSeries struct {
	CadastralNumber string             `json:"cadastral_number"`
	Lat             float64            `json:"lat"`
	Lon             float64            `json:"lon"`
	Intervals       []SatelliteIndices `json:"intervals"`
}

type IndexTrend string

const (
	TrendImproving IndexTrend = "improving"
	TrendDeclining IndexTrend = "declining"
	TrendStable    IndexTrend = "stable"
)

// AgriculturalHealthIndex is a composite score from multiple satellite indices.
type AgriculturalHealthIndex struct {
	Composite   float64 `json:"composite"`     // 0-1 weighted score
	NDVI        float64 `json:"ndvi"`
	NDWINorm    float64 `json:"ndwi_norm"`     // normalized to 0-1
	EVINorm     float64 `json:"evi_norm"`      // normalized to 0-1
	LAINorm     float64 `json:"lai_norm"`      // normalized to 0-1
	WaterStress bool    `json:"water_stress"`
}
