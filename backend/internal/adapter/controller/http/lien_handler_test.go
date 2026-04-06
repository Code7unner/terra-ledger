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
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository/mock"
)

type LienHandlerSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	lienRepo   *mock.MockLienRepo
	parcelRepo *mock.MockParcelRepo
	solana     *mock.MockSolanaClient
	handler    *LienHandler

	cadastral string
}

func TestLienHandlerSuite(t *testing.T) {
	suite.Run(t, new(LienHandlerSuite))
}

func (s *LienHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.lienRepo = mock.NewMockLienRepo(s.ctrl)
	s.parcelRepo = mock.NewMockParcelRepo(s.ctrl)
	s.solana = mock.NewMockSolanaClient(s.ctrl)

	logger := zerolog.Nop()
	s.handler = NewLienHandler(s.lienRepo, s.parcelRepo, s.solana, &logger)
	s.cadastral = fmt.Sprintf("KZ-%s-%03d", gofakeit.LetterN(4), gofakeit.IntRange(1, 999))
}

func (s *LienHandlerSuite) validLienInput() entity.RegisterLienInput {
	return entity.RegisterLienInput{
		CadastralNumber: s.cadastral,
		LenderWallet:    gofakeit.HexUint(64),
		AmountTenge:     int64(gofakeit.IntRange(1000000, 50000000)),
		NotaryCertHash:  gofakeit.HexUint(64),
	}
}

type lienRegisterCase struct {
	name           string
	buildBody      func(s *LienHandlerSuite) []byte
	setupMock      func(s *LienHandlerSuite)
	expectedStatus int
}

func lienRegisterCases() []lienRegisterCase {
	return []lienRegisterCase{
		{
			name: "success",
			buildBody: func(s *LienHandlerSuite) []byte {
				b, _ := json.Marshal(s.validLienInput())
				return b
			},
			setupMock:      setupRegisterSuccess,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "double_pledge_blocked",
			buildBody: func(s *LienHandlerSuite) []byte {
				b, _ := json.Marshal(s.validLienInput())
				return b
			},
			setupMock:      setupRegisterDoublePledge,
			expectedStatus: http.StatusConflict,
		},
		{
			name: "parcel_not_found",
			buildBody: func(s *LienHandlerSuite) []byte {
				b, _ := json.Marshal(s.validLienInput())
				return b
			},
			setupMock:      setupRegisterParcelNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid_json",
			buildBody:      func(_ *LienHandlerSuite) []byte { return []byte("not-json") },
			setupMock:      func(_ *LienHandlerSuite) {},
			expectedStatus: http.StatusBadRequest,
		},
	}
}

func setupRegisterSuccess(s *LienHandlerSuite) {
	s.lienRepo.EXPECT().
		GetActive(gomock.Any(), s.cadastral).
		Return(nil, entity.ErrNotFound).
		Times(1)
	s.parcelRepo.EXPECT().
		GetByCadastral(gomock.Any(), s.cadastral).
		Return(stubParcel(s.cadastral), nil).
		Times(1)
	s.lienRepo.EXPECT().
		Create(gomock.Any(), matchCadastral(s.cadastral)).
		Return(nil).
		Times(1)
}

func setupRegisterDoublePledge(s *LienHandlerSuite) {
	existing := stubLien(s.cadastral, entity.LienStatusActive)
	s.lienRepo.EXPECT().
		GetActive(gomock.Any(), s.cadastral).
		Return(existing, nil).
		Times(1)
}

func setupRegisterParcelNotFound(s *LienHandlerSuite) {
	s.lienRepo.EXPECT().
		GetActive(gomock.Any(), s.cadastral).
		Return(nil, entity.ErrNotFound).
		Times(1)
	s.parcelRepo.EXPECT().
		GetByCadastral(gomock.Any(), s.cadastral).
		Return(nil, entity.ErrNotFound).
		Times(1)
}

func (s *LienHandlerSuite) TestRegister() {
	for _, tc := range lienRegisterCases() {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock(s)

			app := fiber.New()
			app.Post("/liens", s.handler.Register)

			body := tc.buildBody(s)
			req := httptest.NewRequest(http.MethodPost, "/liens", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}

type lienReleaseCase struct {
	name           string
	lienID         string
	setupMock      func(s *LienHandlerSuite, id string)
	expectedStatus int
}

func lienReleaseCases() []lienReleaseCase {
	lenderName := "Halyk Bank"
	return []lienReleaseCase{
		{
			name:   "success",
			lienID: uuid.New().String(),
			setupMock: func(s *LienHandlerSuite, id string) {
				s.lienRepo.EXPECT().
					GetByID(gomock.Any(), id).
					Return(&entity.Encumbrance{
						ID:         uuid.MustParse(id),
						LenderName: lenderName,
						Status:     entity.LienStatusActive,
					}, nil).
					Times(1)
				s.lienRepo.EXPECT().
					UpdateStatus(gomock.Any(), id, entity.LienStatusReleased).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "not_found",
			lienID: uuid.New().String(),
			setupMock: func(s *LienHandlerSuite, id string) {
				s.lienRepo.EXPECT().
					GetByID(gomock.Any(), id).
					Return(nil, entity.ErrNotFound).
					Times(1)
			},
			expectedStatus: http.StatusNotFound,
		},
	}
}

func (s *LienHandlerSuite) TestRelease() {
	for _, tc := range lienReleaseCases() {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock(s, tc.lienID)

			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				c.Locals("lender", &entity.Lender{Name: "Halyk Bank"})
				return c.Next()
			})
			app.Post("/liens/:id/release", s.handler.Release)

			req := httptest.NewRequest(http.MethodPost,
				fmt.Sprintf("/liens/%s/release", tc.lienID), nil)

			resp, err := app.Test(req)
			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}

type lienListCase struct {
	name           string
	setupMock      func(s *LienHandlerSuite)
	expectedStatus int
	expectedLen    int
}

func lienListCases() []lienListCase {
	return []lienListCase{
		{
			name: "returns_list",
			setupMock: func(s *LienHandlerSuite) {
				liens := []entity.Encumbrance{
					*stubLien(s.cadastral, entity.LienStatusActive),
					*stubLien(s.cadastral, entity.LienStatusReleased),
				}
				s.lienRepo.EXPECT().
					ListByParcel(gomock.Any(), s.cadastral).
					Return(liens, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			expectedLen:    2,
		},
		{
			name: "empty_list",
			setupMock: func(s *LienHandlerSuite) {
				s.lienRepo.EXPECT().
					ListByParcel(gomock.Any(), s.cadastral).
					Return([]entity.Encumbrance{}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			expectedLen:    0,
		},
	}
}

func (s *LienHandlerSuite) TestListByParcel() {
	for _, tc := range lienListCases() {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMock(s)

			app := fiber.New()
			app.Get("/parcels/:cadastral/liens", s.handler.ListByParcel)

			req := httptest.NewRequest(http.MethodGet,
				fmt.Sprintf("/parcels/%s/liens", s.cadastral), nil)

			resp, err := app.Test(req)
			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)

			var result []entity.Encumbrance
			s.Require().NoError(json.NewDecoder(resp.Body).Decode(&result))
			s.Len(result, tc.expectedLen)
		})
	}
}
