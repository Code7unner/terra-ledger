package entity

import (
	"time"

	"github.com/google/uuid"
)

type LienStatus string

const (
	LienStatusActive   LienStatus = "active"
	LienStatusReleased LienStatus = "released"
	LienStatusDisputed LienStatus = "disputed"
)

type Encumbrance struct {
	ID              uuid.UUID  `json:"id"`
	ParcelID        uuid.UUID  `json:"parcel_id"`
	CadastralNumber string     `json:"cadastral_number"`
	LenderWallet    string     `json:"lender_wallet"`
	LenderName      string     `json:"lender_name,omitempty"`
	AmountTenge     int64      `json:"amount_tenge"`
	NotaryCertHash  string     `json:"notary_cert_hash,omitempty"`
	OnChainAddress  string     `json:"on_chain_address,omitempty"`
	TxSignature     string     `json:"tx_signature,omitempty"`
	Status          LienStatus `json:"status"`
	RegisteredAt    time.Time  `json:"registered_at"`
	ReleasedAt      *time.Time `json:"released_at,omitempty"`
}

type RegisterLienInput struct {
	CadastralNumber string `json:"cadastral_number"`
	LenderWallet    string `json:"lender_wallet"`
	AmountTenge     int64  `json:"amount_tenge"`
	NotaryCertHash  string `json:"notary_cert_hash"`
}
