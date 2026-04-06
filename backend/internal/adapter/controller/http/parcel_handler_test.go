package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository/mock"
)

type ParcelHandlerSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	parcelRepo *mock.MockParcelRepo
	solana     *mock.MockSolanaClient
	geocoder   *mock.MockGeocoder
	handler    *ParcelHandler

	cadastral string
}

func TestParcelHandlerSuite(t *testing.T) {
	suite.Run(t, new(ParcelHandlerSuite))
}

func (s *ParcelHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.parcelRepo = mock.NewMockParcelRepo(s.ctrl)
	s.solana = mock.NewMockSolanaClient(s.ctrl)
	s.geocoder = mock.NewMockGeocoder(s.ctrl)

	logger := zerolog.Nop()
	s.handler = NewParcelHandler(s.parcelRepo, s.solana, nil, nil, s.geocoder, &logger)
	s.cadastral = fmt.Sprintf("KZ-%s-%03d", gofakeit.LetterN(4), gofakeit.IntRange(1, 999))
}

func (s *ParcelHandlerSuite) validParcelInput() entity.RegisterParcelInput {
	return entity.RegisterParcelInput{
		CadastralNumber: s.cadastral,
		OwnerWallet:     gofakeit.HexUint(64),
		AreaHa:          gofakeit.Float64Range(100, 10000),
		LandClass:       gofakeit.IntRange(1, 5),
		Oblast:          gofakeit.State(),
		Rayon:           gofakeit.City(),
		HolderName:      gofakeit.Name(),
		HolderIINHash:   gofakeit.HexUint(64),
	}
}

type parcelRegisterCase struct {
	name           string
	buildBody      func(s *ParcelHandlerSuite) []byte
	setupMock      func(s *ParcelHandlerSuite)
	expectedStatus int
}

func parcelRegisterCases() []parcelRegisterCase {
	return []parcelRegisterCase{
		{
			name: "success",
			buildBody: func(s *ParcelHandlerSuite) []byte {
				b, _ := json.Marshal(s.validParcelInput())
				return b
			},
			setupMock: func(s *ParcelHandlerSuite) {
				s.geocoder.EXPECT().Resolve(gomock.Any(), gomock.Any()).Return(51.16, 71.47).Times(1)
				s.parcelRepo.EXPECT().
					Create(gomock.Any(), matchCadastral(s.cadastral)).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "duplicate",
			buildBody: func(s *ParcelHandlerSuite) []byte {
				b, _ := json.Marshal(s.validParcelInput())
				return b
			},
			setupMock: func(s *ParcelHandlerSuite) {
				s.geocoder.EXPECT().Resolve(gomock.Any(), gomock.Any()).Return(51.16, 71.47).Times(1)
				s.parcelRepo.EXPECT().
					Create(gomock.Any(), matchCadastral(s.cadastral)).
					Return(entity.ErrAlreadyExists).
					Times(1)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "invalid_json",
			buildBody:      func(_ *ParcelHandlerSuite) []byte { return []byte("not-json") },
			setupMock:      func(_ *ParcelHandlerSuite) {},
			expectedStatus: http.StatusBadRequest,
		},
	}
}

func (s *ParcelHandlerSuite) TestRegister() {
	for _, tc := range parcelRegisterCases() {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock(s)

			app := fiber.New()
			app.Post("/parcels", s.handler.Register)

			body := tc.buildBody(s)
			req := httptest.NewRequest(http.MethodPost, "/parcels", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}

type parcelGetCase struct {
	name           string
	setupMock      func(s *ParcelHandlerSuite)
	expectedStatus int
}

func parcelGetCases() []parcelGetCase {
	return []parcelGetCase{
		{
			name: "found",
			setupMock: func(s *ParcelHandlerSuite) {
				s.parcelRepo.EXPECT().
					GetByCadastral(gomock.Any(), s.cadastral).
					Return(stubParcel(s.cadastral), nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "not_found",
			setupMock: func(s *ParcelHandlerSuite) {
				s.parcelRepo.EXPECT().
					GetByCadastral(gomock.Any(), s.cadastral).
					Return(nil, entity.ErrNotFound).
					Times(1)
			},
			expectedStatus: http.StatusNotFound,
		},
	}
}

func (s *ParcelHandlerSuite) TestGet() {
	for _, tc := range parcelGetCases() {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock(s)

			app := fiber.New()
			app.Get("/parcels/:cadastral", s.handler.Get)

			req := httptest.NewRequest(http.MethodGet,
				fmt.Sprintf("/parcels/%s", s.cadastral), nil)

			resp, err := app.Test(req)
			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)

			if tc.expectedStatus == http.StatusOK {
				var result entity.Parcel
				s.Require().NoError(json.NewDecoder(resp.Body).Decode(&result))
				s.Equal(s.cadastral, result.CadastralNumber)
			}
		})
	}
}
