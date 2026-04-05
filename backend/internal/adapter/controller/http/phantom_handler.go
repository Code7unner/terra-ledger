package http

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// phantomSession stores the Phantom deeplink callback response for relay.
type phantomSession struct {
	CreatedAt time.Time
	// Filled by callback
	PhantomPubKey string
	Nonce         string
	Data          string
	ErrorCode     string
	ErrorMessage  string
	Received      bool
}

// PhantomHandler relays Phantom mobile deeplink responses to desktop sessions.
type PhantomHandler struct {
	sessions sync.Map // map[string]*phantomSession
}

// NewPhantomHandler creates the handler and starts the cleanup goroutine.
func NewPhantomHandler() *PhantomHandler {
	h := &PhantomHandler{}
	go h.cleanup()
	return h
}

// cleanup removes expired sessions every 60 seconds.
func (h *PhantomHandler) cleanup() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		h.sessions.Range(func(key, value any) bool {
			s := value.(*phantomSession)
			if now.Sub(s.CreatedAt) > 5*time.Minute {
				h.sessions.Delete(key)
			}
			return true
		})
	}
}

// CreateSession generates a new relay session ID.
func (h *PhantomHandler) CreateSession(c *fiber.Ctx) error {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate session"})
	}
	sessionID := hex.EncodeToString(b)

	h.sessions.Store(sessionID, &phantomSession{
		CreatedAt: time.Now(),
	})

	return c.JSON(fiber.Map{"session_id": sessionID})
}

// Callback receives the Phantom redirect after user approves/denies connection.
func (h *PhantomHandler) Callback(c *fiber.Ctx) error {
	sessionID := c.Query("session")
	if sessionID == "" {
		return c.Status(400).SendString("Missing session parameter")
	}

	val, ok := h.sessions.Load(sessionID)
	if !ok {
		return c.Status(404).SendString("Session expired or not found")
	}
	s := val.(*phantomSession)

	// Check for error response from Phantom
	if errCode := c.Query("errorCode"); errCode != "" {
		s.ErrorCode = errCode
		s.ErrorMessage = c.Query("errorMessage")
		s.Received = true
		h.sessions.Store(sessionID, s)

		return c.Type("html").SendString(fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta name="viewport" content="width=device-width"><title>TerraLedger</title>
<style>body{background:#0f0f0f;color:#ededed;font-family:Inter,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;text-align:center}
h1{color:#ef4444;font-size:24px}p{color:#a0a0a0;font-size:16px}</style></head>
<body><div><h1>Connection Denied</h1><p>%s</p><p>You can close this tab.</p></div></body></html>`, s.ErrorMessage))
	}

	// Success response
	s.PhantomPubKey = c.Query("phantom_encryption_public_key")
	s.Nonce = c.Query("nonce")
	s.Data = c.Query("data")
	s.Received = true
	h.sessions.Store(sessionID, s)

	return c.Type("html").SendString(`<!DOCTYPE html>
<html><head><meta name="viewport" content="width=device-width"><title>TerraLedger</title>
<style>body{background:#0f0f0f;color:#ededed;font-family:Inter,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;text-align:center}
h1{color:#25d0ab;font-size:24px}p{color:#a0a0a0;font-size:16px}</style></head>
<body><div><h1>Wallet Connected</h1><p>Return to your computer to continue.</p><p>You can close this tab.</p></div></body></html>`)
}

// Poll checks if the Phantom callback has been received for a session.
func (h *PhantomHandler) Poll(c *fiber.Ctx) error {
	sessionID := c.Params("session")
	if sessionID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "missing session"})
	}

	val, ok := h.sessions.Load(sessionID)
	if !ok {
		return c.JSON(fiber.Map{"status": "expired"})
	}
	s := val.(*phantomSession)

	if !s.Received {
		return c.JSON(fiber.Map{"status": "pending"})
	}

	// One-time read: delete after delivering
	h.sessions.Delete(sessionID)

	if s.ErrorCode != "" {
		return c.JSON(fiber.Map{
			"status":        "error",
			"error_code":    s.ErrorCode,
			"error_message": s.ErrorMessage,
		})
	}

	return c.JSON(fiber.Map{
		"status":                       "connected",
		"phantom_encryption_public_key": s.PhantomPubKey,
		"nonce":                        s.Nonce,
		"data":                         s.Data,
	})
}
