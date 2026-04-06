package ndvi

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

const maxWorkers = 3

type UseCase struct {
	provider repository.SatelliteProvider
	geocoder repository.Geocoder
	logger   *zerolog.Logger
}

func New(provider repository.SatelliteProvider, geocoder repository.Geocoder, logger *zerolog.Logger) *UseCase {
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

// FetchAndStoreSeries fetches multi-index time series from the satellite
// provider and persists the resulting certificates via certRepo.
func (uc *UseCase) FetchAndStoreSeries(
	ctx context.Context,
	parcel entity.Parcel,
	months int,
	certRepo repository.CertificateRepo,
) ([]entity.NDVICertificate, error) {
	lat, lon := uc.geocoder.Resolve(parcel.CadastralNumber, parcel.Oblast)
	now := time.Now()
	startDate := now.AddDate(0, -months, 0).Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	ts, err := uc.provider.FetchTimeSeries(ctx, parcel.CadastralNumber, lat, lon, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("fetch time series: %w", err)
	}

	certs := timeSeriesToCerts(ts, parcel)
	if len(certs) == 0 {
		return nil, nil
	}

	if err := certRepo.CreateBatch(ctx, certs); err != nil {
		return nil, fmt.Errorf("store time series certs: %w", err)
	}

	uc.logger.Info().
		Str("cadastral", parcel.CadastralNumber).
		Int("intervals", len(certs)).
		Msg("satellite time series stored")

	return certs, nil
}

func timeSeriesToCerts(ts *entity.SatelliteTimeSeries, parcel entity.Parcel) []entity.NDVICertificate {
	if ts == nil {
		return nil
	}
	certs := make([]entity.NDVICertificate, 0, len(ts.Intervals))
	for _, idx := range ts.Intervals {
		c := indicesToCert(idx, parcel)
		certs = append(certs, c)
	}
	return certs
}

func indicesToCert(idx entity.SatelliteIndices, parcel entity.Parcel) entity.NDVICertificate {
	ndwi := idx.NDWI
	evi := idx.EVI
	lai := idx.LAI
	cloudFree := idx.CloudFreePct
	sampleCount := idx.SampleCount

	return entity.NDVICertificate{
		ParcelID:        parcel.ID,
		CadastralNumber: parcel.CadastralNumber,
		Season:          seasonLabel(idx.PeriodStart),
		NDVIScore:       idx.NDVI,
		NDWIScore:       &ndwi,
		EVIScore:        &evi,
		LAIEstimate:     &lai,
		CloudFreePct:    &cloudFree,
		SampleCount:     &sampleCount,
		ObservedAt:      idx.PeriodStart,
	}
}

func seasonLabel(t time.Time) string {
	q := (t.Month()-1)/3 + 1
	return fmt.Sprintf("%d-Q%d", t.Year(), q)
}

// ComputeIndexTrend determines if a series of index values is improving,
// declining, or stable by comparing the first and last thirds.
func ComputeIndexTrend(values []float64) entity.IndexTrend {
	if len(values) < 2 {
		return entity.TrendStable
	}

	third := len(values) / 3
	if third == 0 {
		third = 1
	}

	earlyAvg := avg(values[:third])
	lateAvg := avg(values[len(values)-third:])

	const threshold = 0.03
	diff := lateAvg - earlyAvg
	if diff > threshold {
		return entity.TrendImproving
	}
	if diff < -threshold {
		return entity.TrendDeclining
	}
	return entity.TrendStable
}

// ComputeWaterStressRisk returns true when the average NDWI indicates drought.
func ComputeWaterStressRisk(avgNDWI float64) bool {
	return avgNDWI < -0.3
}

func avg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// ComputeAHI calculates the Agricultural Health Index from certificates.
// Weights: NDVI 40%, EVI 25%, NDWI 20%, LAI 15%.
func ComputeAHI(certs []entity.NDVICertificate) *entity.AgriculturalHealthIndex {
	if len(certs) == 0 {
		return nil
	}

	var ndviSum, ndwiSum, eviSum, laiSum float64
	var ndwiCount, eviCount, laiCount int

	for _, c := range certs {
		ndviSum += c.NDVIScore
		if c.NDWIScore != nil {
			ndwiSum += *c.NDWIScore
			ndwiCount++
		}
		if c.EVIScore != nil {
			eviSum += *c.EVIScore
			eviCount++
		}
		if c.LAIEstimate != nil {
			laiSum += *c.LAIEstimate
			laiCount++
		}
	}

	avgNDVI := ndviSum / float64(len(certs))

	ndwiNorm := normalizeNDWI(ndwiSum, ndwiCount)
	eviNorm := normalizeEVI(eviSum, eviCount)
	laiNorm := normalizeLAI(laiSum, laiCount)

	composite := 0.40*avgNDVI + 0.25*eviNorm + 0.20*ndwiNorm + 0.15*laiNorm

	return &entity.AgriculturalHealthIndex{
		Composite:   composite,
		NDVI:        avgNDVI,
		NDWINorm:    ndwiNorm,
		EVINorm:     eviNorm,
		LAINorm:     laiNorm,
		WaterStress: ndwiCount > 0 && ComputeWaterStressRisk(ndwiSum/float64(ndwiCount)),
	}
}

// normalizeNDWI converts NDWI from [-1,1] to [0,1].
func normalizeNDWI(sum float64, count int) float64 {
	if count == 0 {
		return 0.5
	}
	return (sum/float64(count) + 1) / 2
}

// normalizeEVI converts EVI from [0, 0.8] to [0, 1].
func normalizeEVI(sum float64, count int) float64 {
	if count == 0 {
		return 0.5
	}
	norm := sum / float64(count) / 0.8
	if norm > 1 {
		norm = 1
	}
	return norm
}

// normalizeLAI converts LAI from [0, 8] to [0, 1].
func normalizeLAI(sum float64, count int) float64 {
	if count == 0 {
		return 0.5
	}
	norm := sum / float64(count) / 8.0
	if norm > 1 {
		norm = 1
	}
	return norm
}
