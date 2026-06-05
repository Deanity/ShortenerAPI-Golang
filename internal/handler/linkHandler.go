package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"shortenerapi/internal/domain"
	"shortenerapi/pkg/utils"
)

type LinkHandler struct {
	linkUseCase domain.LinkUseCase
}

func NewLinkHandler(linkUseCase domain.LinkUseCase) *LinkHandler {
	return &LinkHandler{
		linkUseCase: linkUseCase,
	}
}

type shortenRequest struct {
	OriginalURL string     `json:"original_url"`
	Slug        string     `json:"slug"`
	Tags        []string   `json:"tags"`
	WebhookURL  string     `json:"webhook_url"`
	Password    string     `json:"password"`
	ExpiresAt   *time.Time `json:"expires_at"`
	ClickLimit  *int       `json:"click_limit"`
}

type bulkShortenRequest struct {
	URLs []string `json:"urls"`
}

type updateLinkRequest struct {
	OriginalURL string     `json:"original_url"`
	Tags        []string   `json:"tags"`
	IsActive    *bool      `json:"is_active"`
	WebhookURL  string     `json:"webhook_url"`
	ExpiresAt   *time.Time `json:"expires_at"`
	ClickLimit  *int       `json:"click_limit"`
}

func (h *LinkHandler) Shorten(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	var req shortenRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid request body", "INVALID_BODY")
	}
	if req.OriginalURL == "" {
		return utils.Error(c, fiber.StatusBadRequest, "original_url is required", "MISSING_URL")
	}

	options := map[string]interface{}{
		"slug":        req.Slug,
		"tags":        req.Tags,
		"webhook_url": req.WebhookURL,
		"password":    req.Password,
		"expires_at":  req.ExpiresAt,
		"click_limit": req.ClickLimit,
	}

	link, err := h.linkUseCase.Shorten(c.Context(), user.ID, req.OriginalURL, options)
	if err != nil {
		switch err {
		case domain.ErrSlugTaken:
			return utils.Error(c, fiber.StatusConflict, "Slug already taken", "SLUG_TAKEN")
		case domain.ErrUnsafeURL:
			return utils.Error(c, fiber.StatusUnprocessableEntity, "URL flagged as unsafe by Safe Browsing", "UNSAFE_URL")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, "Failed to create link", "CREATE_FAILED")
		}
	}

	return utils.Success(c, fiber.StatusCreated, "Link created successfully", link)
}

func (h *LinkHandler) BulkShorten(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	var req bulkShortenRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid request body", "INVALID_BODY")
	}
	if len(req.URLs) == 0 {
		return utils.Error(c, fiber.StatusBadRequest, "urls array is required", "MISSING_URLS")
	}
	if len(req.URLs) > 100 {
		return utils.Error(c, fiber.StatusBadRequest, "Maximum 100 URLs per bulk request", "TOO_MANY_URLS")
	}

	links, err := h.linkUseCase.BulkShorten(c.Context(), user.ID, req.URLs, nil)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "Bulk shorten failed", "BULK_FAILED")
	}

	return utils.Success(c, fiber.StatusCreated, "Links created successfully", links)
}

func (h *LinkHandler) Get(c *fiber.Ctx) error {
	_, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid link ID", "INVALID_ID")
	}

	link, err := h.linkUseCase.GetLink(c.Context(), id)
	if err != nil {
		switch err {
		case domain.ErrLinkNotFound:
			return utils.Error(c, fiber.StatusNotFound, "Link not found", "LINK_NOT_FOUND")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, "Failed to get link", "FETCH_FAILED")
		}
	}

	return utils.Success(c, fiber.StatusOK, "Link fetched", link)
}

func (h *LinkHandler) Update(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid link ID", "INVALID_ID")
	}

	var req updateLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid request body", "INVALID_BODY")
	}

	updates := map[string]interface{}{}
	if req.OriginalURL != "" {
		updates["original_url"] = req.OriginalURL
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.WebhookURL != "" {
		updates["webhook_url"] = req.WebhookURL
	}
	if req.ExpiresAt != nil {
		updates["expires_at"] = req.ExpiresAt
	}
	if req.ClickLimit != nil {
		updates["click_limit"] = req.ClickLimit
	}

	link, err := h.linkUseCase.UpdateLink(c.Context(), id, user.ID, updates)
	if err != nil {
		switch err {
		case domain.ErrLinkNotFound:
			return utils.Error(c, fiber.StatusNotFound, "Link not found", "LINK_NOT_FOUND")
		case domain.ErrUnauthorized:
			return utils.Error(c, fiber.StatusForbidden, "You do not own this link", "FORBIDDEN")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, "Failed to update link", "UPDATE_FAILED")
		}
	}

	return utils.Success(c, fiber.StatusOK, "Link updated successfully", link)
}

func (h *LinkHandler) Delete(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid link ID", "INVALID_ID")
	}

	if err := h.linkUseCase.DeleteLink(c.Context(), id, user.ID); err != nil {
		switch err {
		case domain.ErrLinkNotFound:
			return utils.Error(c, fiber.StatusNotFound, "Link not found", "LINK_NOT_FOUND")
		case domain.ErrUnauthorized:
			return utils.Error(c, fiber.StatusForbidden, "You do not own this link", "FORBIDDEN")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, "Failed to delete link", "DELETE_FAILED")
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *LinkHandler) List(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	tag := c.Query("tag")

	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		b := v == "true"
		isActive = &b
	}

	links, total, err := h.linkUseCase.ListLinks(c.Context(), user.ID, tag, isActive, page, perPage)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "Failed to list links", "LIST_FAILED")
	}

	return utils.Paginated(c, fiber.StatusOK, "Links fetched", links, page, perPage, int64(total))
}
