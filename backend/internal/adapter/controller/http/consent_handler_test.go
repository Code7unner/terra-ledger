package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository/mock"
)

type ConsentHandlerSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	consentRepo *mock.MockConsentRepo
	handler     *ConsentHandler

	wallet string
}

func TestConsentHandlerSuite(t *testing.T) {
	suite.Run(t, new(ConsentHandlerSuite))
}

func (s *ConsentHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.consentRepo = mock.NewMockConsentRepo(s.ctrl)
	s.handler = NewConsentHandler(s.consentRepo)
	s.wallet = gofakeit.HexUint(64)
}

func (s *ConsentHandlerSuite) setupApp() *fiber.App {
	app := fiber.New()
	app.Post("/consent/grant", s.handler.Grant)
	app.Post("/consent/revoke", s.handler.Revoke)
	app.Get("/consent/:wallet", s.handler.Get)
	app.Get("/consent/:wallet/log", s.handler.ListAccessLog)
	return app
}

func stubConsent(wallet string, status entity.ConsentStatus) *entity.Consent {
	now := time.Now()
	c := &entity.Consent{
		ID:            uuid.New(),
		WalletAddress: wallet,
		Status:        status,
		CreatedAt:     now.Add(-24 * time.Hour),
	}
	if status == entity.ConsentStatusGranted {
		c.GrantedAt = &now
	}
	if status == entity.ConsentStatusRevoked {
		c.RevokedAt = &now
	}
	return c
}

func stubConsentLogEntry(lenderName, dataType string) entity.ConsentLogEntry {
	return entity.ConsentLogEntry{
		ID:           uuid.New(),
		ConsentID:    uuid.New(),
		LenderWallet: gofakeit.HexUint(64),
		LenderName:   lenderName,
		DataType:     dataType,
		AccessedAt:   time.Now().Add(-time.Duration(gofakeit.IntRange(1, 48)) * time.Hour),
	}
}

type consentTestCase struct {
	name string

	method      string
	buildPath   func(wallet string) string
	sendBody    bool
	bodyWallet  func(wallet string) string
	emptyWallet bool

	setupMock      func(s *ConsentHandlerSuite, wallet string)
	expectedStatus int
}

func (s *ConsentHandlerSuite) TestConsentOperations() {
	cases := []consentTestCase{
		{
			name:       "grant_new_consent",
			method:     http.MethodPost,
			buildPath:  func(_ string) string { return "/consent/grant" },
			sendBody:   true,
			bodyWallet: func(w string) string { return w },
			setupMock: func(s *ConsentHandlerSuite, wallet string) {
				s.consentRepo.EXPECT().
					Grant(gomock.Any(), wallet).
					Return(stubConsent(wallet, entity.ConsentStatusGranted), nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "grant_missing_wallet",
			method:      http.MethodPost,
			buildPath:   func(_ string) string { return "/consent/grant" },
			sendBody:    true,
			emptyWallet: true,
			bodyWallet:  func(_ string) string { return "" },
			setupMock:   func(_ *ConsentHandlerSuite, _ string) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "revoke_active_consent",
			method:     http.MethodPost,
			buildPath:  func(_ string) string { return "/consent/revoke" },
			sendBody:   true,
			bodyWallet: func(w string) string { return w },
			setupMock: func(s *ConsentHandlerSuite, wallet string) {
				s.consentRepo.EXPECT().
					Revoke(gomock.Any(), wallet).
					Return(stubConsent(wallet, entity.ConsentStatusRevoked), nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "revoke_not_found",
			method:     http.MethodPost,
			buildPath:  func(_ string) string { return "/consent/revoke" },
			sendBody:   true,
			bodyWallet: func(w string) string { return w },
			setupMock: func(s *ConsentHandlerSuite, wallet string) {
				s.consentRepo.EXPECT().
					Revoke(gomock.Any(), wallet).
					Return(nil, entity.ErrNotFound).
					Times(1)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "get_consent",
			method:   http.MethodGet,
			buildPath: func(w string) string { return fmt.Sprintf("/consent/%s", w) },
			setupMock: func(s *ConsentHandlerSuite, wallet string) {
				s.consentRepo.EXPECT().
					GetByWallet(gomock.Any(), wallet).
					Return(stubConsent(wallet, entity.ConsentStatusGranted), nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "get_consent_not_found",
			method:   http.MethodGet,
			buildPath: func(w string) string { return fmt.Sprintf("/consent/%s", w) },
			setupMock: func(s *ConsentHandlerSuite, wallet string) {
				s.consentRepo.EXPECT().
					GetByWallet(gomock.Any(), wallet).
					Return(nil, entity.ErrNotFound).
					Times(1)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "list_access_log",
			method:   http.MethodGet,
			buildPath: func(w string) string { return fmt.Sprintf("/consent/%s/log", w) },
			setupMock: func(s *ConsentHandlerSuite, wallet string) {
				entries := []entity.ConsentLogEntry{
					stubConsentLogEntry("Halyk Bank", "credit_score"),
					stubConsentLogEntry("Kaspi", "parcel_info"),
				}
				s.consentRepo.EXPECT().
					ListAccessLog(gomock.Any(), wallet).
					Return(entries, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			app := s.setupApp()

			tc.setupMock(s, s.wallet)

			var req *http.Request
			if tc.sendBody {
				walletVal := s.wallet
				if tc.bodyWallet != nil {
					walletVal = tc.bodyWallet(s.wallet)
				}
				body, _ := json.Marshal(map[string]string{"wallet_address": walletVal})
				req = httptest.NewRequest(tc.method, tc.buildPath(s.wallet), bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.buildPath(s.wallet), nil)
			}

			resp, err := app.Test(req)
			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}
