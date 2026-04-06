package entity

import (
	"time"

	"github.com/google/uuid"
)

type NDVICertificate struct {
	ID              uuid.UUID `json:"id"`
	ParcelID        uuid.UUID `json:"parcel_id"`
	CadastralNumber string    `json:"cadastral_number"`
	Season          string    `json:"season"`
	NDVIScore       float64   `json:"ndvi_score"`
	NDWIScore       *float64  `json:"ndwi_score,omitempty"`
	EVIScore        *float64  `json:"evi_score,omitempty"`
	LAIEstimate     *float64  `json:"lai_estimate,omitempty"`
	CloudFreePct    *float64  `json:"cloud_free_pct,omitempty"`
	SampleCount     *int      `json:"sample_count,omitempty"`
	CropType        string    `json:"crop_type,omitempty"`
	YieldTHa        float64   `json:"yield_t_ha,omitempty"`
	SentinelSceneID string    `json:"sentinel_scene_id,omitempty"`
	OnChainAddress  string    `json:"on_chain_address,omitempty"`
	TxSignature     string    `json:"tx_signature,omitempty"`
	ObservedAt      time.Time `json:"observed_at,omitempty"`
	MintedAt        time.Time `json:"minted_at"`
}
