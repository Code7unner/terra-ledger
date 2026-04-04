package entity

import (
	"time"

	"github.com/google/uuid"
)

type ConsentStatus string

const (
	ConsentStatusPending ConsentStatus = "pending"
	ConsentStatusGranted ConsentStatus = "granted"
	ConsentStatusRevoked ConsentStatus = "revoked"
)

type Consent struct {
	ID            uuid.UUID     `json:"id"`
	WalletAddress string        `json:"wallet_address"`
	Status        ConsentStatus `json:"status"`
	GrantedAt     *time.Time    `json:"granted_at,omitempty"`
	RevokedAt     *time.Time    `json:"revoked_at,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

type ConsentLogEntry struct {
	ID           uuid.UUID `json:"id"`
	ConsentID    uuid.UUID `json:"consent_id"`
	LenderWallet string    `json:"lender_wallet"`
	LenderName   string    `json:"lender_name,omitempty"`
	DataType     string    `json:"data_type"`
	AccessedAt   time.Time `json:"accessed_at"`
}
