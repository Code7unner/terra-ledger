package repository

import (
	"context"
	"fmt"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
)

// EGISSMock returns fixture-based parcel snapshot data for known cadastral numbers.
// It implements the repository.EGISSOracle interface.
type EGISSMock struct{}

func NewEGISSMock() *EGISSMock {
	return &EGISSMock{}
}

var fixtures = map[string]map[string]any{
	"01:001:0001": {
		"cadastral_number": "01:001:0001",
		"oblast":           "Akmola",
		"rayon":            "Tselinograd",
		"area_ha":          250.5,
		"land_class":       3,
		"holder_name":      "Nursultan Akhmetov",
		"holder_iin_hash":  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		"status":           "active",
		"registered_date":  "2020-03-15",
	},
	"01:002:0015": {
		"cadastral_number": "01:002:0015",
		"oblast":           "Kostanay",
		"rayon":            "Karabalyk",
		"area_ha":          480.0,
		"land_class":       2,
		"holder_name":      "Aigul Serikova",
		"holder_iin_hash":  "b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3",
		"status":           "active",
		"registered_date":  "2018-07-22",
	},
	"03:010:0042": {
		"cadastral_number": "03:010:0042",
		"oblast":           "North Kazakhstan",
		"rayon":            "Kyzylzhar",
		"area_ha":          120.0,
		"land_class":       4,
		"holder_name":      "Marat Zhumabekov",
		"holder_iin_hash":  "c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
		"status":           "active",
		"registered_date":  "2021-11-01",
	},
}

func (m *EGISSMock) GetParcelSnapshot(ctx context.Context, cadastral string) (map[string]any, error) {
	snap, ok := fixtures[cadastral]
	if !ok {
		return nil, fmt.Errorf("cadastral %q: %w", cadastral, entity.ErrNotFound)
	}

	// Return a copy to prevent mutation of the fixture data.
	result := make(map[string]any, len(snap))
	for k, v := range snap {
		result[k] = v
	}

	return result, nil
}
