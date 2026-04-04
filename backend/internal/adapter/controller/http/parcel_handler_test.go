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

type ParcelHandlerSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	parcelRepo *mock.MockParcelRepo
	handler    *ParcelHandler
}

func TestParcelHandlerSuite(t *testing.T) {
	suite.Run(t, new(ParcelHandlerSuite))
}

func (s *ParcelHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.parcelRepo = mock.NewMockParcelRepo(s.ctrl)
	s.handler = NewParcelHandler(s.parcelRepo, nil, nil)
}

func (s *ParcelHandlerSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *ParcelHandlerSuite) setupApp() *fiber.App {
	app := fiber.New()
	app.Post("/parcels", s.handler.Register)
	app.Get("/parcels/:cadastral", s.handler.Get)
	return app
}

func (s *ParcelHandlerSuite) TestParcelHandler_Register() {
	cadastral := gofakeit.LetterN(12)

	tests := []struct {
		name           string
		body           func() []byte
		mockSetup      func()
		expectedStatus int
	}{
		{
			name: "register success",
			body: func() []byte {
				input := entity.RegisterParcelInput{
					CadastralNumber: cadastral,
					OwnerWallet:     gofakeit.HexUint(64),
					AreaHa:          gofakeit.Float64Range(100, 10000),
					LandClass:       gofakeit.IntRange(1, 5),
					Oblast:          gofakeit.State(),
					Rayon:           gofakeit.City(),
					HolderName:      gofakeit.Name(),
					HolderIINHash:   gofakeit.HexUint(64),
				}
				b, _ := json.Marshal(input)
				return b
			},
			mockSetup: func() {
				s.parcelRepo.EXPECT().
					Create(gomock.Any(), matchCadastral(cadastral)).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "register duplicate",
			body: func() []byte {
				input := entity.RegisterParcelInput{
					CadastralNumber: cadastral,
					OwnerWallet:     gofakeit.HexUint(64),
					AreaHa:          gofakeit.Float64Range(100, 10000),
					LandClass:       gofakeit.IntRange(1, 5),
					Oblast:          gofakeit.State(),
					Rayon:           gofakeit.City(),
					HolderName:      gofakeit.Name(),
					HolderIINHash:   gofakeit.HexUint(64),
				}
				b, _ := json.Marshal(input)
				return b
			},
			mockSetup: func() {
				s.parcelRepo.EXPECT().
					Create(gomock.Any(), matchCadastral(cadastral)).
					Return(entity.ErrAlreadyExists).
					Times(1)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "register bad body",
			body: func() []byte {
				return []byte(`{invalid-json`)
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.mockSetup()

			app := s.setupApp()
			req := httptest.NewRequest(http.MethodPost, "/parcels", bytes.NewReader(tc.body()))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)

			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}

func (s *ParcelHandlerSuite) TestParcelHandler_Get() {
	cadastral := gofakeit.LetterN(12)

	tests := []struct {
		name           string
		cadastral      string
		mockSetup      func()
		expectedStatus int
	}{
		{
			name:      "get success",
			cadastral: cadastral,
			mockSetup: func() {
				s.parcelRepo.EXPECT().
					GetByCadastral(gomock.Any(), cadastral).
					Return(stubParcel(cadastral), nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "get not found",
			cadastral: cadastral,
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
			req := httptest.NewRequest(http.MethodGet, "/parcels/"+tc.cadastral, nil)

			resp, err := app.Test(req)

			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}
