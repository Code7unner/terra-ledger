package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	mock "github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository/mock"
)

type LienHandlerSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	lienRepo   *mock.MockLienRepo
	parcelRepo *mock.MockParcelRepo
	handler    *LienHandler
}

func TestLienHandlerSuite(t *testing.T) {
	suite.Run(t, new(LienHandlerSuite))
}

func (s *LienHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.lienRepo = mock.NewMockLienRepo(s.ctrl)
	s.parcelRepo = mock.NewMockParcelRepo(s.ctrl)
	s.handler = NewLienHandler(s.lienRepo, s.parcelRepo, nil, nil)
}

func (s *LienHandlerSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *LienHandlerSuite) setupApp() *fiber.App {
	app := fiber.New()
	app.Post("/liens", s.handler.Register)
	app.Post("/liens/:id/release", s.handler.Release)
	app.Get("/parcels/:cadastral/liens", s.handler.ListByParcel)
	return app
}

func (s *LienHandlerSuite) TestLienHandler_Register() {
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
				input := entity.RegisterLienInput{
					CadastralNumber: cadastral,
					LenderWallet:    gofakeit.HexUint(64),
					AmountTenge:     int64(gofakeit.IntRange(1000000, 50000000)),
					NotaryCertHash:  gofakeit.HexUint(64),
				}
				b, _ := json.Marshal(input)
				return b
			},
			mockSetup: func() {
				parcel := stubParcel(cadastral)

				s.lienRepo.EXPECT().
					GetActive(gomock.Any(), cadastral).
					Return(nil, entity.ErrNotFound).
					Times(1)
				s.parcelRepo.EXPECT().
					GetByCadastral(gomock.Any(), cadastral).
					Return(parcel, nil).
					Times(1)
				s.lienRepo.EXPECT().
					Create(gomock.Any(), matchCadastral(cadastral)).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "register double pledge",
			body: func() []byte {
				input := entity.RegisterLienInput{
					CadastralNumber: cadastral,
					LenderWallet:    gofakeit.HexUint(64),
					AmountTenge:     int64(gofakeit.IntRange(1000000, 50000000)),
					NotaryCertHash:  gofakeit.HexUint(64),
				}
				b, _ := json.Marshal(input)
				return b
			},
			mockSetup: func() {
				existing := stubLien(cadastral, entity.LienStatusActive)

				s.lienRepo.EXPECT().
					GetActive(gomock.Any(), cadastral).
					Return(existing, nil).
					Times(1)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "register parcel not found",
			body: func() []byte {
				input := entity.RegisterLienInput{
					CadastralNumber: cadastral,
					LenderWallet:    gofakeit.HexUint(64),
					AmountTenge:     int64(gofakeit.IntRange(1000000, 50000000)),
					NotaryCertHash:  gofakeit.HexUint(64),
				}
				b, _ := json.Marshal(input)
				return b
			},
			mockSetup: func() {
				s.lienRepo.EXPECT().
					GetActive(gomock.Any(), cadastral).
					Return(nil, entity.ErrNotFound).
					Times(1)
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
			req := httptest.NewRequest(http.MethodPost, "/liens", bytes.NewReader(tc.body()))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)

			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}

func (s *LienHandlerSuite) TestLienHandler_Release() {
	lienID := uuid.New().String()

	tests := []struct {
		name           string
		id             string
		mockSetup      func()
		expectedStatus int
	}{
		{
			name: "release success",
			id:   lienID,
			mockSetup: func() {
				s.lienRepo.EXPECT().
					UpdateStatus(gomock.Any(), lienID, matchLienStatus(entity.LienStatusReleased)).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "release not found",
			id:   lienID,
			mockSetup: func() {
				s.lienRepo.EXPECT().
					UpdateStatus(gomock.Any(), lienID, matchLienStatus(entity.LienStatusReleased)).
					Return(entity.ErrNotFound).
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
			req := httptest.NewRequest(http.MethodPost, "/liens/"+tc.id+"/release", nil)

			resp, err := app.Test(req)

			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}

func (s *LienHandlerSuite) TestLienHandler_ListByParcel() {
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
				liens := []entity.Encumbrance{
					*stubLien(cadastral, entity.LienStatusActive),
					*stubLien(cadastral, entity.LienStatusReleased),
				}
				s.lienRepo.EXPECT().
					ListByParcel(gomock.Any(), cadastral).
					Return(liens, nil).
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
			req := httptest.NewRequest(http.MethodGet, "/parcels/"+tc.cadastral+"/liens", nil)

			resp, err := app.Test(req)

			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}
