package service

import (
	"context"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository/mock"
)

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
			name: "one_parcel_needs_check",
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
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock(s, tc)

			logger := zerolog.Nop()
			k := &Keeper{
				solana:            s.solana,
				parcels:           s.parcelRepo,
				interval:          6 * time.Hour,
				terraTokenProgram: "2eAqpJ7yjso7FDA4sDQLJQioNCRuoYSUeha2Y88NRRMX",
				logger:            &logger,
			}

			k.processTick(context.Background())
			// No panics = success. Mock expectations verified by gomock.
		})
	}
}
