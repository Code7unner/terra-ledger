package entity

import (
	"time"

	"github.com/google/uuid"
)

type Lender struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	BIN              string    `json:"bin,omitempty"`
	APIKey           string    `json:"-"`
	Tier             string    `json:"tier"`
	QueriesThisMonth int       `json:"queries_this_month"`
	QueryLimit       int       `json:"query_limit"`
	CreatedAt        time.Time `json:"created_at"`
}
