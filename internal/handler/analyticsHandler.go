package handler

import (
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"shortenerapi/internal/domain"
	"shortenerapi/pkg/utils"
)

type AnalyticsHandler struct {
	analyticsUseCase domain.AnalyticsUseCase
}

func NewAnalyticsHandler(analyticsUseCase domain.AnalyticsUseCase) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsUseCase: analyticsUseCase,
	}
}

func (h *AnalyticsHandler) GetAnalytics(c *fiber.Ctx) error {
	_, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid link ID", "INVALID_ID")
	}

	analytics, err := h.analyticsUseCase.GetAnalytics(c.Context(), id)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "Failed to fetch analytics", "FETCH_FAILED")
	}

	return utils.Success(c, fiber.StatusOK, "Analytics fetched", analytics)
}
