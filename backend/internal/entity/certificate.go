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
	CropType        string    `json:"crop_type,omitempty"`
	YieldTHa        float64   `json:"yield_t_ha,omitempty"`
	SentinelSceneID string    `json:"sentinel_scene_id,omitempty"`
	OnChainAddress  string    `json:"on_chain_address,omitempty"`
	TxSignature     string    `json:"tx_signature,omitempty"`
	MintedAt        time.Time `json:"minted_at"`
}
