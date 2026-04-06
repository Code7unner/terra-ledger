package entity

import (
	"time"

	"github.com/google/uuid"
)

type Parcel struct {
	ID               uuid.UUID  `json:"id"`
	CadastralNumber  string     `json:"cadastral_number"`
	OwnerWallet      string     `json:"owner_wallet"`
	OnChainAddress   string     `json:"on_chain_address,omitempty"`
	AreaHa           float64    `json:"area_ha"`
	LandClass        int        `json:"land_class"`
	KYCVerified      bool       `json:"kyc_verified"`
	Oblast           string     `json:"oblast,omitempty"`
	Rayon            string     `json:"rayon,omitempty"`
	HolderName       string     `json:"holder_name,omitempty"`
	HolderIINHash    string     `json:"holder_iin_hash,omitempty"`
	EGISSSnapshot    any        `json:"egiss_snapshot,omitempty"`
	RegisteredAt     time.Time  `json:"registered_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Latitude         *float64   `json:"latitude,omitempty"`
	Longitude        *float64   `json:"longitude,omitempty"`
}

type RegisterParcelInput struct {
	CadastralNumber string  `json:"cadastral_number"`
	OwnerWallet     string  `json:"owner_wallet"`
	AreaHa          float64 `json:"area_ha"`
	LandClass       int     `json:"land_class"`
	Oblast          string  `json:"oblast"`
	Rayon           string  `json:"rayon"`
	HolderName      string  `json:"holder_name"`
	HolderIINHash   string  `json:"holder_iin_hash"`
}
