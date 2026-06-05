package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"shortenerapi/internal/domain"
	"shortenerapi/pkg/utils"
)


type AuthHandler struct {
	authUseCase domain.AuthUseCase
}

func NewAuthHandler(authUseCase domain.AuthUseCase) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
	}
}

type registerRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type createAPIKeyRequest struct {
	Label  string   `json:"label" validate:"required,min=1,max=100"`
	Scopes []string `json:"scopes"`
	TeamID *string  `json:"team_id"`
}

type deleteAPIKeyRequest struct {
	Label string `json:"label" validate:"required"`
}

type customDomainRequest struct {
	Domain string `json:"domain" validate:"required"`
}


func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid request body", "INVALID_BODY")
	}

	user, err := h.authUseCase.Register(c.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		switch err {
		case domain.ErrUserAlreadyExists:
			return utils.Error(c, fiber.StatusConflict, "Email already registered", "EMAIL_TAKEN")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, "Registration failed", "REGISTRATION_FAILED")
		}
	}

	return utils.Success(c, fiber.StatusCreated, "User registered successfully", user)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid request body", "INVALID_BODY")
	}

	token, err := h.authUseCase.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		switch err {
		case domain.ErrInvalidCredentials:
			return utils.Error(c, fiber.StatusUnauthorized, "Invalid email or password", "INVALID_CREDENTIALS")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, "Login failed", "LOGIN_FAILED")
		}
	}

	return utils.Success(c, fiber.StatusOK, "Login successful", fiber.Map{"token": token})
}

func (h *AuthHandler) CreateAPIKey(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	var req createAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid request body", "INVALID_BODY")
	}

	plaintext, err := h.authUseCase.CreateAPIKey(c.Context(), user.ID, req.Label, req.Scopes, req.TeamID)
	if err != nil {
		log.Error().Err(err).Msg("CreateAPIKey failed")
		return utils.Error(c, fiber.StatusInternalServerError, "Failed to create API key", "API_KEY_CREATION_FAILED")
	}

	return utils.Success(c, fiber.StatusCreated, "API key created successfully", fiber.Map{
		"key":     plaintext,
		"label":   req.Label,
		"scopes":  req.Scopes,
		"team_id": req.TeamID,
	})
}



func (h *AuthHandler) ListAPIKeys(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	keys, err := h.authUseCase.ListAPIKeys(c.Context(), user.ID)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "Failed to fetch API keys", "FETCH_FAILED")
	}

	return utils.Success(c, fiber.StatusOK, "API keys fetched", keys)
}

func (h *AuthHandler) DeleteAPIKey(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	var req deleteAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid request body", "INVALID_BODY")
	}

	if err := h.authUseCase.RevokeAPIKey(c.Context(), user.ID, req.Label); err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "Failed to revoke API key", "REVOKE_FAILED")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *AuthHandler) AddCustomDomain(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	var req customDomainRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid request body", "INVALID_BODY")
	}
	if req.Domain == "" {
		return utils.Error(c, fiber.StatusBadRequest, "Domain is required", "MISSING_DOMAIN")
	}

	if err := h.authUseCase.AddCustomDomain(c.Context(), user.ID, req.Domain); err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "Failed to add custom domain", "ADD_DOMAIN_FAILED")
	}

	return utils.Success(c, fiber.StatusCreated, "Custom domain added successfully", fiber.Map{"domain": req.Domain})
}

func (h *AuthHandler) DeleteCustomDomain(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	var req customDomainRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid request body", "INVALID_BODY")
	}
	if req.Domain == "" {
		return utils.Error(c, fiber.StatusBadRequest, "Domain is required", "MISSING_DOMAIN")
	}

	if err := h.authUseCase.DeleteCustomDomain(c.Context(), user.ID, req.Domain); err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "Failed to delete custom domain", "DELETE_DOMAIN_FAILED")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *AuthHandler) ListCustomDomains(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*domain.User)
	if !ok {
		return utils.Error(c, fiber.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
	}

	domains, err := h.authUseCase.ListCustomDomains(c.Context(), user.ID)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "Failed to fetch custom domains", "FETCH_DOMAINS_FAILED")
	}

	return utils.Success(c, fiber.StatusOK, "Custom domains fetched", domains)
}

