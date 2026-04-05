package ndvi

import (
	"context"
	"sync"

	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

const maxWorkers = 3

type UseCase struct {
	provider repository.NDVIProvider
	geocoder repository.Geocoder
	logger   *zerolog.Logger
}

func New(provider repository.NDVIProvider, geocoder repository.Geocoder, logger *zerolog.Logger) *UseCase {
	return &UseCase{provider: provider, geocoder: geocoder, logger: logger}
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

			lat, lon := uc.geocoder.Resolve(parcel.CadastralNumber, parcel.Oblast)
			score, err := uc.provider.FetchNDVI(ctx, parcel.CadastralNumber, lat, lon, "", "")
			if err != nil {
				uc.logger.Warn().
					Err(err).
					Str("cadastral", parcel.CadastralNumber).
					Msg("NDVI fetch failed")
				return
			}

			mu.Lock()
			results[parcel.CadastralNumber] = score
			mu.Unlock()

			uc.logger.Info().
				Str("cadastral", parcel.CadastralNumber).
				Float64("ndvi", score).
				Msg("NDVI fetched")
		}(p)
	}

	wg.Wait()
	return results
}
