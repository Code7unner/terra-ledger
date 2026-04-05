package service

import (
	"context"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/gagliardetto/solana-go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository/mock"
)

const testTerraTokenProgram = "2eAqpJ7yjso7FDA4sDQLJQioNCRuoYSUeha2Y88NRRMX"

// fakeBlockhash is a valid base58-encoded 32-byte hash for test usage.
const fakeBlockhash = "4sGjMW1sUnHzSxGspuhpqLDx6wiyjNtZAMdL4VZHirAn"

type KeeperSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	parcelRepo *mock.MockParcelRepo
	solana     *mock.MockSolanaClient
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(KeeperSuite))
}

func (s *KeeperSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.parcelRepo = mock.NewMockParcelRepo(s.ctrl)
	s.solana = mock.NewMockSolanaClient(s.ctrl)
}

func (s *KeeperSuite) newKeeper() *Keeper {
	logger := zerolog.Nop()
	relayKey := solana.NewWallet().PrivateKey

	return &Keeper{
		solana:            s.solana,
		parcels:           s.parcelRepo,
		interval:          6 * time.Hour,
		relayKey:          relayKey,
		terraTokenProgram: testTerraTokenProgram,
		logger:            &logger,
	}
}

type keeperTestCase struct {
	name string

	parcels     []entity.Parcel
	parcelsErr  error
	setupMock   func(s *KeeperSuite, tc keeperTestCase)
	expectError bool
}

func (s *KeeperSuite) TestProcessTick() {
	cases := []keeperTestCase{
		{
			name:    "no_parcels_need_check",
			parcels: nil,
			setupMock: func(s *KeeperSuite, _ keeperTestCase) {
				s.parcelRepo.EXPECT().
					ListNeedingSeasonalCheck(gomock.Any(), gomock.Any()).
					Return(nil, nil).
					Times(1)
			},
		},
		{
			name: "sends_transaction_on_success",
			parcels: []entity.Parcel{
				{
					CadastralNumber: gofakeit.LetterN(12),
					OnChainAddress:  gofakeit.HexUint(64),
				},
			},
			setupMock: func(s *KeeperSuite, tc keeperTestCase) {
				s.parcelRepo.EXPECT().
					ListNeedingSeasonalCheck(gomock.Any(), gomock.Any()).
					Return(tc.parcels, nil).
					Times(1)
				s.solana.EXPECT().
					GetRecentBlockhash(gomock.Any()).
					Return(fakeBlockhash, nil).
					Times(1)
				s.solana.EXPECT().
					SendTransaction(gomock.Any(), gomock.Any()).
					Return(gofakeit.LetterN(88), nil).
					Times(1)
			},
		},
		{
			name:       "repo_query_fails",
			parcelsErr: gofakeit.Error(),
			setupMock: func(s *KeeperSuite, tc keeperTestCase) {
				s.parcelRepo.EXPECT().
					ListNeedingSeasonalCheck(gomock.Any(), gomock.Any()).
					Return(nil, tc.parcelsErr).
					Times(1)
			},
		},
		{
			name: "blockhash_fetch_fails",
			parcels: []entity.Parcel{
				{CadastralNumber: gofakeit.LetterN(12)},
			},
			setupMock: func(s *KeeperSuite, tc keeperTestCase) {
				s.parcelRepo.EXPECT().
					ListNeedingSeasonalCheck(gomock.Any(), gomock.Any()).
					Return(tc.parcels, nil).
					Times(1)
				s.solana.EXPECT().
					GetRecentBlockhash(gomock.Any()).
					Return("", gofakeit.Error()).
					Times(1)
			},
		},
		{
			name: "send_transaction_fails",
			parcels: []entity.Parcel{
				{CadastralNumber: gofakeit.LetterN(12)},
			},
			setupMock: func(s *KeeperSuite, tc keeperTestCase) {
				s.parcelRepo.EXPECT().
					ListNeedingSeasonalCheck(gomock.Any(), gomock.Any()).
					Return(tc.parcels, nil).
					Times(1)
				s.solana.EXPECT().
					GetRecentBlockhash(gomock.Any()).
					Return(fakeBlockhash, nil).
					Times(1)
				s.solana.EXPECT().
					SendTransaction(gomock.Any(), gomock.Any()).
					Return("", gofakeit.Error()).
					Times(1)
			},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock(s, tc)

			k := s.newKeeper()
			k.processTick(context.Background())
			// No panics = success. Mock expectations verified by gomock.
		})
	}
}
