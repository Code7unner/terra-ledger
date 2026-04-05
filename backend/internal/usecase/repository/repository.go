package repository

import (
	"context"
	"time"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
)

type ParcelRepo interface {
	Create(ctx context.Context, p *entity.Parcel) error
	GetByCadastral(ctx context.Context, cadastral string) (*entity.Parcel, error)
	UpdateOnChainAddress(ctx context.Context, cadastral, addr string) error
	ListNeedingSeasonalCheck(ctx context.Context, maxAge time.Duration) ([]entity.Parcel, error)
}

type CertificateRepo interface {
	Create(ctx context.Context, cert *entity.NDVICertificate) error
	ListByParcel(ctx context.Context, cadastral string) ([]entity.NDVICertificate, error)
	GetLatest(ctx context.Context, cadastral string) (*entity.NDVICertificate, error)
}

type LienRepo interface {
	Create(ctx context.Context, lien *entity.Encumbrance) error
	GetActive(ctx context.Context, cadastral string) (*entity.Encumbrance, error)
	ListByParcel(ctx context.Context, cadastral string) ([]entity.Encumbrance, error)
	UpdateStatus(ctx context.Context, id string, status entity.LienStatus) error
	FindByWalletAndCadastral(ctx context.Context, lenderWallet, cadastral string) (*entity.Encumbrance, error)
}

type LenderRepo interface {
	GetByAPIKey(ctx context.Context, key string) (*entity.Lender, error)
}

type CreditScoreRepo interface {
	Upsert(ctx context.Context, score *entity.CreditScore) error
	GetByCadastral(ctx context.Context, cadastral string) (*entity.CreditScore, error)
}

type SignatureRepo interface {
	SignatureExists(ctx context.Context, signature string) (bool, error)
	RecordSignature(ctx context.Context, signature, programID string) error
}

type SolanaClient interface {
	GetAccountInfo(ctx context.Context, address string) ([]byte, error)
	GetRecentBlockhash(ctx context.Context) (string, error)
	SendTransaction(ctx context.Context, txBytes []byte) (string, error)
	SimulateTransaction(ctx context.Context, txBytes []byte) error
	GetSignaturesForAddress(ctx context.Context, address string, limit int) ([]string, error)
}

type NDVIProvider interface {
	FetchNDVI(ctx context.Context, cadastral string, lat, lon float64, startDate, endDate string) (float64, error)
}

type EGISSOracle interface {
	GetParcelSnapshot(ctx context.Context, cadastral string) (map[string]any, error)
}

type CreditScorer interface {
	ComputeScore(ctx context.Context, input *entity.ScoringInput) (*entity.CreditScore, error)
}

type ConsentRepo interface {
	Grant(ctx context.Context, walletAddress string) (*entity.Consent, error)
	Revoke(ctx context.Context, walletAddress string) (*entity.Consent, error)
	GetByWallet(ctx context.Context, walletAddress string) (*entity.Consent, error)
	LogAccess(ctx context.Context, entry *entity.ConsentLogEntry) error
	ListAccessLog(ctx context.Context, walletAddress string) ([]entity.ConsentLogEntry, error)
}
