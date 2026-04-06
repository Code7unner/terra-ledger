package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/code7unner/decentrathon5/terra-ledger/backend/internal/usecase/repository"
)

type AgentHandler struct {
	decisionRepo repository.AgentDecisionRepo
}

func NewAgentHandler(decisionRepo repository.AgentDecisionRepo) *AgentHandler {
	return &AgentHandler{decisionRepo: decisionRepo}
}

func (h *AgentHandler) ListRecent(c *fiber.Ctx) error {
	decisions, err := h.decisionRepo.ListRecent(c.Context(), 50)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch decisions",
		})
	}

	return c.JSON(fiber.Map{"decisions": decisions})
}

func (h *AgentHandler) ListByParcel(c *fiber.Ctx) error {
	cadastral := c.Params("cadastral")

	decisions, err := h.decisionRepo.ListByParcel(c.Context(), cadastral, 20)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch decisions",
		})
	}

	return c.JSON(fiber.Map{"decisions": decisions})
}
