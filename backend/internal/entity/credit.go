package entity

import (
	"time"

	"github.com/google/uuid"
)

type CreditScore struct {
	ID                 uuid.UUID `json:"id"`
	ParcelID           uuid.UUID `json:"parcel_id"`
	CadastralNumber    string    `json:"cadastral_number"`
	AIScore            int       `json:"ai_score"`
	RecommendedLTV     float64   `json:"recommended_ltv"`
	CollateralGrade    string    `json:"collateral_grade"`
	EstimatedValueTenge int64    `json:"estimated_value_tenge"`
	ModelVersion       string    `json:"model_version"`
	Explanation        string    `json:"explanation"`
	RiskFactors        []string  `json:"risk_factors"`
	ComputedAt         time.Time `json:"computed_at"`
}

type ScoringInput struct {
	CadastralNumber string
	AreaHa          float64
	LandClass       int
	Oblast          string
	NDVIHistory     []NDVICertificate
	ActiveLiens     int
	TotalLiens      int
	Disputes        int
	DormantSeasons  int
}

type CreditProfile struct {
	Parcel       Parcel            `json:"parcel"`
	Productivity ProductivityData  `json:"productivity"`
	Encumbrances EncumbranceData   `json:"encumbrances"`
	Credit       *CreditScore      `json:"credit_intelligence,omitempty"`
}

type ProductivityData struct {
	Certificates []NDVICertificate `json:"certificates"`
	NDVITrend    string            `json:"ndvi_trend"`
	DormancyRisk string            `json:"dormancy_risk"`
}

type EncumbranceData struct {
	ActiveLiens        []Encumbrance `json:"active_liens"`
	LienCountHistorical int          `json:"lien_count_historical"`
	DoublePledgeRisk   bool          `json:"double_pledge_risk"`
}
