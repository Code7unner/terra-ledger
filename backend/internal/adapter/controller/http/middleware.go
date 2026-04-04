package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/entity"
	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

func APIKeyAuth(lenderRepo repository.LenderRepo, logger *zerolog.Logger) fiber.Handler {
	// TODO: Phase 3
	return func(c *fiber.Ctx) error {
		key := c.Get("X-API-Key")
		if key == "" {
			auth := c.Get("Authorization")
			key = strings.TrimPrefix(auth, "Bearer ")
		}

		if key == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": entity.ErrUnauthorized.Error(),
			})
		}

		lender, err := lenderRepo.GetByAPIKey(c.Context(), key)
		if err != nil {
			logger.Warn().Str("key_prefix", key[:min(8, len(key))]).Msg("invalid API key")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": entity.ErrUnauthorized.Error(),
			})
		}

		c.Locals("lender", lender)
		return c.Next()
	}
}

func RequestLogger(logger *zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		logger.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Msg("request")
		return err
	}
}
