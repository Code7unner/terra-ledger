package http

import (
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
)

func stubParcel(cadastral string) *entity.Parcel {
	return &entity.Parcel{
		ID:              uuid.New(),
		CadastralNumber: cadastral,
		OwnerWallet:     gofakeit.HexUint(64),
		AreaHa:          gofakeit.Float64Range(100, 10000),
		LandClass:       gofakeit.IntRange(1, 5),
		KYCVerified:     true,
		Oblast:          gofakeit.State(),
		Rayon:           gofakeit.City(),
		HolderName:      gofakeit.Name(),
		RegisteredAt:    time.Now().Add(-24 * time.Hour),
		UpdatedAt:       time.Now(),
	}
}

func stubCert(cadastral, season string) *entity.NDVICertificate {
	return &entity.NDVICertificate{
		ID:              uuid.New(),
		ParcelID:        uuid.New(),
		CadastralNumber: cadastral,
		Season:          season,
		NDVIScore:       gofakeit.Float64Range(0.3, 0.9),
		CropType:        gofakeit.RandomString([]string{"wheat", "barley", "sunflower"}),
		YieldTHa:        gofakeit.Float64Range(1.0, 5.0),
		MintedAt:        time.Now().Add(-48 * time.Hour),
	}
}

func stubLien(cadastral string, status entity.LienStatus) *entity.Encumbrance {
	return &entity.Encumbrance{
		ID:              uuid.New(),
		ParcelID:        uuid.New(),
		CadastralNumber: cadastral,
		LenderWallet:    gofakeit.HexUint(64),
		LenderName:      gofakeit.Company(),
		AmountTenge:     int64(gofakeit.IntRange(1000000, 50000000)),
		NotaryCertHash:  gofakeit.HexUint(64),
		Status:          status,
		RegisteredAt:    time.Now().Add(-24 * time.Hour),
	}
}

func stubCreditScore(cadastral string, aiScore int) *entity.CreditScore {
	grade := "D"
	ltv := 0.1
	switch {
	case aiScore >= 80:
		grade, ltv = "A", 0.7
	case aiScore >= 60:
		grade, ltv = "B", 0.5
	case aiScore >= 40:
		grade, ltv = "C", 0.3
	}

	return &entity.CreditScore{
		ID:                  uuid.New(),
		CadastralNumber:     cadastral,
		AIScore:             aiScore,
		RecommendedLTV:      ltv,
		CollateralGrade:     grade,
		EstimatedValueTenge: int64(gofakeit.IntRange(5000000, 100000000)),
		ModelVersion:        "test-v1",
		Explanation:         gofakeit.Sentence(10),
		RiskFactors:         []string{gofakeit.Sentence(5)},
		ComputedAt:          time.Now(),
	}
}

func stubStaleCreditScore(cadastral string, aiScore int) *entity.CreditScore {
	score := stubCreditScore(cadastral, aiScore)
	score.ComputedAt = time.Now().Add(-2 * time.Hour)
	return score
}

// matchCadastral verifies entity has expected cadastral_number field.
// Verified fields: CadastralNumber (exact match)
func matchCadastral(expected string) gomock.Matcher {
	return gomock.Cond(func(x any) bool {
		switch v := x.(type) {
		case *entity.Parcel:
			return v.CadastralNumber == expected
		case *entity.NDVICertificate:
			return v.CadastralNumber == expected
		case *entity.Encumbrance:
			return v.CadastralNumber == expected
		default:
			return false
		}
	})
}

// matchLienStatus verifies encumbrance has expected status.
// Verified fields: Status (exact match)
func matchLienStatus(expected entity.LienStatus) gomock.Matcher {
	return gomock.Cond(func(x any) bool {
		status, ok := x.(entity.LienStatus)
		return ok && status == expected
	})
}

// matchScoringInput verifies ScoringInput has expected cadastral and lien counts.
// Verified fields: CadastralNumber (exact), ActiveLiens (exact), TotalLiens (exact)
func matchScoringInput(cadastral string, activeLiens, totalLiens int) gomock.Matcher {
	return gomock.Cond(func(x any) bool {
		input, ok := x.(*entity.ScoringInput)
		if !ok {
			return false
		}
		return input.CadastralNumber == cadastral &&
			input.ActiveLiens == activeLiens &&
			input.TotalLiens == totalLiens
	})
}
