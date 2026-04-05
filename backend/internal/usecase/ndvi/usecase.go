package ndvi

import (
	"context"
	"sync"

	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

const maxWorkers = 3

// Approximate centroids for Akmola oblast parcels.
var parcelCentroids = map[string][2]float64{
	"KZ11-0032-001": {51.1283, 69.4120},
	"KZ11-0032-002": {51.1350, 69.4200},
	"KZ11-0032-003": {51.1400, 69.4050},
	"KZ11-0032-004": {51.1180, 69.3980},
	"KZ11-0032-005": {51.1500, 69.4300},
}

type UseCase struct {
	provider repository.NDVIProvider
	logger   *zerolog.Logger
}

func New(provider repository.NDVIProvider, logger *zerolog.Logger) *UseCase {
	return &UseCase{provider: provider, logger: logger}
}

func (uc *UseCase) ProcessBatch(ctx context.Context, parcels []entity.Parcel) map[string]float64 {
	results := make(map[string]float64)
	var mu sync.Mutex
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, p := range parcels {
		select {
		case <-ctx.Done():
			return results
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(parcel entity.Parcel) {
			defer wg.Done()
			defer func() { <-sem }()

			lat, lon := CentroidFor(parcel.CadastralNumber)
			ndvi, err := uc.provider.FetchNDVI(ctx, parcel.CadastralNumber, lat, lon, "", "")
			if err != nil {
				uc.logger.Warn().
					Err(err).
					Str("cadastral", parcel.CadastralNumber).
					Msg("NDVI fetch failed")
				return
			}

			mu.Lock()
			results[parcel.CadastralNumber] = ndvi
			mu.Unlock()

			uc.logger.Info().
				Str("cadastral", parcel.CadastralNumber).
				Float64("ndvi", ndvi).
				Msg("NDVI fetched")
		}(p)
	}

	wg.Wait()
	return results
}

// CentroidFor returns lat/lon for a cadastral number.
func CentroidFor(cadastral string) (float64, float64) {
	if c, ok := parcelCentroids[cadastral]; ok {
		return c[0], c[1]
	}
	return 51.13, 69.41 // default Akmola oblast
}
