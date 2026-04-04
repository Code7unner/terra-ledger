package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	mock "github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository/mock"
)

type CertificateHandlerSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	certRepo   *mock.MockCertificateRepo
	parcelRepo *mock.MockParcelRepo
	handler    *CertificateHandler
}

func TestCertificateHandlerSuite(t *testing.T) {
	suite.Run(t, new(CertificateHandlerSuite))
}

func (s *CertificateHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.certRepo = mock.NewMockCertificateRepo(s.ctrl)
	s.parcelRepo = mock.NewMockParcelRepo(s.ctrl)
	s.handler = NewCertificateHandler(s.certRepo, s.parcelRepo)
}

func (s *CertificateHandlerSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *CertificateHandlerSuite) setupApp() *fiber.App {
	app := fiber.New()
	app.Post("/parcels/:cadastral/certificates", s.handler.Mint)
	app.Get("/parcels/:cadastral/certificates", s.handler.List)
	return app
}

func (s *CertificateHandlerSuite) TestCertificateHandler_Mint() {
	cadastral := gofakeit.LetterN(12)
	season := gofakeit.RandomString([]string{"2025-spring", "2025-autumn", "2026-spring"})

	tests := []struct {
		name           string
		cadastral      string
		body           func() []byte
		mockSetup      func()
		expectedStatus int
	}{
		{
			name:      "mint success",
			cadastral: cadastral,
			body: func() []byte {
				input := map[string]any{
					"season":            season,
					"ndvi_score":        gofakeit.Float64Range(0.3, 0.9),
					"crop_type":         gofakeit.RandomString([]string{"wheat", "barley", "sunflower"}),
					"yield_t_ha":        gofakeit.Float64Range(1.0, 5.0),
					"sentinel_scene_id": gofakeit.LetterN(20),
				}
				b, _ := json.Marshal(input)
				return b
			},
			mockSetup: func() {
				parcel := stubParcel(cadastral)

				s.parcelRepo.EXPECT().
					GetByCadastral(gomock.Any(), cadastral).
					Return(parcel, nil).
					Times(1)
				s.certRepo.EXPECT().
					Create(gomock.Any(), matchCadastral(cadastral)).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:      "mint parcel not found",
			cadastral: cadastral,
			body: func() []byte {
				input := map[string]any{
					"season":            season,
					"ndvi_score":        gofakeit.Float64Range(0.3, 0.9),
					"crop_type":         gofakeit.RandomString([]string{"wheat", "barley", "sunflower"}),
					"yield_t_ha":        gofakeit.Float64Range(1.0, 5.0),
					"sentinel_scene_id": gofakeit.LetterN(20),
				}
				b, _ := json.Marshal(input)
				return b
			},
			mockSetup: func() {
				s.parcelRepo.EXPECT().
					GetByCadastral(gomock.Any(), cadastral).
					Return(nil, entity.ErrNotFound).
					Times(1)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.mockSetup()

			app := s.setupApp()
			req := httptest.NewRequest(
				http.MethodPost,
				"/parcels/"+tc.cadastral+"/certificates",
				bytes.NewReader(tc.body()),
			)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)

			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}

func (s *CertificateHandlerSuite) TestCertificateHandler_List() {
	cadastral := gofakeit.LetterN(12)

	tests := []struct {
		name           string
		cadastral      string
		mockSetup      func()
		expectedStatus int
	}{
		{
			name:      "list success",
			cadastral: cadastral,
			mockSetup: func() {
				certs := []entity.NDVICertificate{
					*stubCert(cadastral, "2025-spring"),
					*stubCert(cadastral, "2025-autumn"),
				}
				s.certRepo.EXPECT().
					ListByParcel(gomock.Any(), cadastral).
					Return(certs, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.mockSetup()

			app := s.setupApp()
			req := httptest.NewRequest(
				http.MethodGet,
				"/parcels/"+tc.cadastral+"/certificates",
				nil,
			)

			resp, err := app.Test(req)

			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}
