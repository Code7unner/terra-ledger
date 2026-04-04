package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
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

type WebhookHandlerSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	parcelRepo *mock.MockParcelRepo
	certRepo   *mock.MockCertificateRepo
	lienRepo   *mock.MockLienRepo
	handler    *WebhookHandler

	secret    string
	cadastral string
}

func TestWebhookHandlerSuite(t *testing.T) {
	suite.Run(t, new(WebhookHandlerSuite))
}

func (s *WebhookHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.parcelRepo = mock.NewMockParcelRepo(s.ctrl)
	s.certRepo = mock.NewMockCertificateRepo(s.ctrl)
	s.lienRepo = mock.NewMockLienRepo(s.ctrl)
	s.secret = gofakeit.UUID()
	s.cadastral = gofakeit.LetterN(12)

	logger := zerolog.Nop()
	s.handler = NewWebhookHandler(
		s.secret,
		"2eAqpJ7yjso7FDA4sDQLJQioNCRuoYSUeha2Y88NRRMX",
		"3qYHSTPeRLRDfWmtzEhiaHpT2kchgW8GqaYcwmDbKnq4",
		s.parcelRepo, s.certRepo, s.lienRepo,
		&logger,
	)
}

func (s *WebhookHandlerSuite) setupApp() *fiber.App {
	app := fiber.New()
	app.Post("/webhooks/helius", s.handler.Handle)
	return app
}

func buildEventLog(eventName string, payload []byte) string {
	h := sha256.Sum256([]byte("event:" + eventName))
	data := make([]byte, 8+len(payload))
	copy(data[:8], h[:8])
	copy(data[8:], payload)
	return "Program data: " + base64.StdEncoding.EncodeToString(data)
}

func encodeBorshString(s string) []byte {
	b := make([]byte, 4+len(s))
	binary.LittleEndian.PutUint32(b[:4], uint32(len(s)))
	copy(b[4:], s)
	return b
}

func encodePubkeyBytes() []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(gofakeit.IntRange(0, 255))
	}
	return b
}

func encodeU16LE(v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return b
}

func encodeU32LE(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func encodeU64LE(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

func stubHeliusTx(sig string, logMessages []string) HeliusTransaction {
	return HeliusTransaction{
		Signature: sig,
		Type:      "UNKNOWN",
		Meta:      HeliusMeta{LogMessages: logMessages},
	}
}

type webhookTestCase struct {
	name string

	wrongAuth bool
	badJSON   bool

	buildPayload func(cadastral string) []HeliusTransaction
	setupMock    func(s *WebhookHandlerSuite)

	expectedStatus int
}

func (s *WebhookHandlerSuite) TestWebhookHandle() {
	cases := []webhookTestCase{
		{
			name: "valid_ParcelRegistered_event",
			buildPayload: func(cadastral string) []HeliusTransaction {
				var payload []byte
				payload = append(payload, encodeBorshString(cadastral)...)
				payload = append(payload, encodePubkeyBytes()...)
				payload = append(payload, encodeU32LE(4530)...)
				log := buildEventLog("ParcelRegistered", payload)
				return []HeliusTransaction{stubHeliusTx(gofakeit.UUID(), []string{log})}
			},
			setupMock: func(s *WebhookHandlerSuite) {
				s.parcelRepo.EXPECT().
					UpdateOnChainAddress(gomock.Any(), s.cadastral, gomock.Any()).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid_CertificateMinted_event",
			buildPayload: func(cadastral string) []HeliusTransaction {
				var payload []byte
				payload = append(payload, encodeBorshString(cadastral)...)
				payload = append(payload, encodeBorshString("2026-Q1")...)
				payload = append(payload, encodeU16LE(750)...)
				payload = append(payload, encodePubkeyBytes()...)
				log := buildEventLog("CertificateMinted", payload)
				return []HeliusTransaction{stubHeliusTx(gofakeit.UUID(), []string{log})}
			},
			setupMock: func(s *WebhookHandlerSuite) {
				s.certRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid_EncumbranceRegistered_event",
			buildPayload: func(cadastral string) []HeliusTransaction {
				var payload []byte
				payload = append(payload, encodeBorshString(cadastral)...)
				payload = append(payload, encodePubkeyBytes()...)
				payload = append(payload, encodeU64LE(15000000)...)
				payload = append(payload, encodePubkeyBytes()...)
				log := buildEventLog("EncumbranceRegistered", payload)
				return []HeliusTransaction{stubHeliusTx(gofakeit.UUID(), []string{log})}
			},
			setupMock: func(s *WebhookHandlerSuite) {
				s.lienRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid_EncumbranceReleased_event",
			buildPayload: func(cadastral string) []HeliusTransaction {
				var payload []byte
				payload = append(payload, encodeBorshString(cadastral)...)
				payload = append(payload, encodePubkeyBytes()...)
				payload = append(payload, encodePubkeyBytes()...)
				log := buildEventLog("EncumbranceReleased", payload)
				return []HeliusTransaction{stubHeliusTx(gofakeit.UUID(), []string{log})}
			},
			setupMock: func(s *WebhookHandlerSuite) {
				existing := stubLien(s.cadastral, entity.LienStatusActive)
				s.lienRepo.EXPECT().
					FindByWalletAndCadastral(gomock.Any(), gomock.Any(), s.cadastral).
					Return(existing, nil).
					Times(1)
				s.lienRepo.EXPECT().
					UpdateStatus(gomock.Any(), existing.ID.String(), entity.LienStatusReleased).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid_auth_header",
			wrongAuth:      true,
			buildPayload:   func(_ string) []HeliusTransaction { return nil },
			setupMock:      func(_ *WebhookHandlerSuite) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "malformed_payload",
			badJSON:        true,
			buildPayload:   func(_ string) []HeliusTransaction { return nil },
			setupMock:      func(_ *WebhookHandlerSuite) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "no_matching_events",
			buildPayload: func(_ string) []HeliusTransaction {
				return []HeliusTransaction{stubHeliusTx(
					gofakeit.UUID(),
					[]string{"Program log: some other log"},
				)}
			},
			setupMock:      func(_ *WebhookHandlerSuite) {},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			app := s.setupApp()
			tc.setupMock(s)

			var bodyBytes []byte
			if tc.badJSON {
				bodyBytes = []byte("not-json")
			} else {
				txns := tc.buildPayload(s.cadastral)
				bodyBytes, _ = json.Marshal(txns)
			}

			req := httptest.NewRequest(http.MethodPost, "/webhooks/helius", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			if tc.wrongAuth {
				req.Header.Set("Authorization", "wrong-secret")
			} else {
				req.Header.Set("Authorization", s.secret)
			}

			resp, err := app.Test(req)
			s.Require().NoError(err)
			s.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}
