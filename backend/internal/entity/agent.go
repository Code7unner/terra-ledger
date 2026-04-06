package entity

import (
	"time"

	"github.com/google/uuid"
)

type AgentDecision struct {
	ID              uuid.UUID `json:"id"`
	ParcelID        uuid.UUID `json:"parcel_id"`
	CadastralNumber string    `json:"cadastral_number"`
	PreviousScore   *int      `json:"previous_score,omitempty"`
	NewScore        int       `json:"new_score"`
	PreviousGrade   *string   `json:"previous_grade,omitempty"`
	NewGrade        string    `json:"new_grade"`
	Reason          string    `json:"reason"`
	TxSignature     string    `json:"tx_signature,omitempty"`
	DecidedAt       time.Time `json:"decided_at"`
}
