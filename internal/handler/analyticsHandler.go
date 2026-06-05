package handler

import (
	"time"

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

func (h *AnalyticsHandler) GetGeoAnalytics(c *fiber.Ctx) error {
	_, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid link ID", "INVALID_ID")
	}
	data, err := h.analyticsUseCase.GetGeoAnalytics(c.Context(), id)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "Failed to fetch geo analytics", "FETCH_FAILED")
	}
	return utils.Success(c, fiber.StatusOK, "Geo analytics fetched", data)
}

func (h *AnalyticsHandler) GetDeviceAnalytics(c *fiber.Ctx) error {
	_, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid link ID", "INVALID_ID")
	}
	data, err := h.analyticsUseCase.GetDeviceAnalytics(c.Context(), id)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "Failed to fetch device analytics", "FETCH_FAILED")
	}
	return utils.Success(c, fiber.StatusOK, "Device analytics fetched", data)
}

func (h *AnalyticsHandler) GetReferrerAnalytics(c *fiber.Ctx) error {
	_, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid link ID", "INVALID_ID")
	}
	data, err := h.analyticsUseCase.GetReferrerAnalytics(c.Context(), id)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "Failed to fetch referrer analytics", "FETCH_FAILED")
	}
	return utils.Success(c, fiber.StatusOK, "Referrer analytics fetched", data)
}

func (h *AnalyticsHandler) GetTimeSeriesAnalytics(c *fiber.Ctx) error {
	_, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid link ID", "INVALID_ID")
	}

	// Parse optional query params
	interval := c.Query("interval", "day") // hour | day | week | month
	startStr := c.Query("start")
	endStr := c.Query("end")

	now := time.Now()
	start := now.AddDate(0, -1, 0) // default: last 30 days
	end := now

	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}

	data, err := h.analyticsUseCase.GetTimeSeriesAnalytics(c.Context(), id, start, end, interval)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "Failed to fetch time-series analytics", "FETCH_FAILED")
	}
	return utils.Success(c, fiber.StatusOK, "Time-series analytics fetched", data)
}
