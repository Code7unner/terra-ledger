package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository/mock"
)

type CreditHandlerSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	parcelRepo *mock.MockParcelRepo
	certRepo   *mock.MockCertificateRepo
	lienRepo   *mock.MockLienRepo
	scoreRepo  *mock.MockCreditScoreRepo
	scorer     *mock.MockCreditScorer
	handler    *CreditHandler

	cadastral string
}

func TestCreditHandlerSuite(t *testing.T) {
	suite.Run(t, new(CreditHandlerSuite))
}

func (s *CreditHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.parcelRepo = mock.NewMockParcelRepo(s.ctrl)
	s.certRepo = mock.NewMockCertificateRepo(s.ctrl)
	s.lienRepo = mock.NewMockLienRepo(s.ctrl)
	s.scoreRepo = mock.NewMockCreditScoreRepo(s.ctrl)
	s.scorer = mock.NewMockCreditScorer(s.ctrl)
	s.handler = NewCreditHandler(s.parcelRepo, s.certRepo, s.lienRepo, s.scoreRepo, nil, s.scorer)

	s.cadastral = fmt.Sprintf("KZ-%s-%03d", gofakeit.LetterN(4), gofakeit.IntRange(1, 999))
}

func (s *CreditHandlerSuite) setupApp() *fiber.App {
	app := fiber.New()
	app.Get("/parcels/:cadastral/profile", s.handler.GetProfile)
	return app
}

type creditTestCase struct {
	name string

	parcelExists   bool
	hasCerts       bool
	hasActiveLien  bool
	cachedScore    bool
	cachedScoreAge time.Duration
	scorerFails    bool

	expectedStatus int
	scorerCalled   bool
	upsertCalled   bool
}

func (s *CreditHandlerSuite) TestGetProfile() {
	cases := []creditTestCase{
		{
			name:           "fresh_score_exists",
			parcelExists:   true,
			hasCerts:       true,
			cachedScore:    true,
			cachedScoreAge: 30 * time.Minute,
			expectedStatus: http.StatusOK,
			scorerCalled:   false,
			upsertCalled:   false,
		},
		{
			name:           "stale_score_triggers_recompute",
			parcelExists:   true,
			hasCerts:       true,
			cachedScore:    true,
			cachedScoreAge: 2 * time.Hour,
			expectedStatus: http.StatusOK,
			scorerCalled:   true,
			upsertCalled:   true,
		},
		{
			name:           "no_cached_score_triggers_compute",
			parcelExists:   true,
			hasCerts:       true,
			cachedScore:    false,
			expectedStatus: http.StatusOK,
			scorerCalled:   true,
			upsertCalled:   true,
		},
		{
			name:           "scorer_fails_returns_stale",
			parcelExists:   true,
			hasCerts:       true,
			cachedScore:    true,
			cachedScoreAge: 2 * time.Hour,
			scorerFails:    true,
			expectedStatus: http.StatusOK,
			scorerCalled:   true,
			upsertCalled:   false,
		},
		{
			name:           "parcel_not_found",
			parcelExists:   false,
			expectedStatus: http.StatusNotFound,
			scorerCalled:   false,
			upsertCalled:   false,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			app := s.setupApp()

			parcel := stubParcel(s.cadastral)
			certs := []entity.NDVICertificate{*stubCert(s.cadastral, "2025-Q3")}
			liens := []entity.Encumbrance{}
			if tc.hasActiveLien {
				liens = append(liens, *stubLien(s.cadastral, entity.LienStatusActive))
			}

			if !tc.parcelExists {
				s.parcelRepo.EXPECT().
					GetByCadastral(gomock.Any(), s.cadastral).
					Return(nil, entity.ErrNotFound).
					Times(1)
			} else {
				s.parcelRepo.EXPECT().
					GetByCadastral(gomock.Any(), s.cadastral).
					Return(parcel, nil).
					Times(1)

				if tc.hasCerts {
					s.certRepo.EXPECT().
						ListByParcel(gomock.Any(), s.cadastral).
						Return(certs, nil).
						Times(1)
				} else {
					s.certRepo.EXPECT().
						ListByParcel(gomock.Any(), s.cadastral).
						Return(nil, nil).
						Times(1)
				}

				s.lienRepo.EXPECT().
					ListByParcel(gomock.Any(), s.cadastral).
					Return(liens, nil).
					Times(1)

				if tc.cachedScore {
					score := stubCreditScore(s.cadastral, gofakeit.IntRange(40, 90))
					score.ComputedAt = time.Now().Add(-tc.cachedScoreAge)
					s.scoreRepo.EXPECT().
						GetByCadastral(gomock.Any(), s.cadastral).
						Return(score, nil).
						Times(1)
				} else {
					s.scoreRepo.EXPECT().
						GetByCadastral(gomock.Any(), s.cadastral).
						Return(nil, entity.ErrNotFound).
						Times(1)
				}

				if tc.scorerCalled {
					if tc.scorerFails {
						s.scorer.EXPECT().
							ComputeScore(gomock.Any(), matchScoringInput(s.cadastral, 0, len(liens))).
							Return(nil, fmt.Errorf("scorer unavailable")).
							Times(1)
					} else {
						freshScore := stubCreditScore(s.cadastral, gofakeit.IntRange(50, 95))
						s.scorer.EXPECT().
							ComputeScore(gomock.Any(), matchScoringInput(s.cadastral, 0, len(liens))).
							Return(freshScore, nil).
							Times(1)

						if tc.upsertCalled {
							s.scoreRepo.EXPECT().
								Upsert(gomock.Any(), gomock.Any()).
								Return(nil).
								Times(1)
						}
					}
				}
			}

			req := httptest.NewRequest(http.MethodGet,
				fmt.Sprintf("/parcels/%s/profile", s.cadastral), nil)
			resp, err := app.Test(req)

			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}
