package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"shortenerapi/internal/domain"
	"shortenerapi/pkg/utils"
)

type RedirectHandler struct {
	linkUseCase      domain.LinkUseCase
	analyticsUseCase domain.AnalyticsUseCase
}

func NewRedirectHandler(linkUseCase domain.LinkUseCase, analyticsUseCase domain.AnalyticsUseCase) *RedirectHandler {
	return &RedirectHandler{
		linkUseCase:      linkUseCase,
		analyticsUseCase: analyticsUseCase,
	}
}

type unlockRequest struct {
	Password string `json:"password"`
}

func (h *RedirectHandler) Redirect(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")
	clientIP := c.IP()
	userAgent := c.Get("User-Agent")
	referrer := c.Get("Referer")

	targetURL, link, err := h.linkUseCase.ResolveRedirect(c.Context(), shortCode, "", clientIP, userAgent, referrer)
	if err != nil {
		switch err {
		case domain.ErrLinkNotFound:
			return utils.Error(c, fiber.StatusNotFound, "Short link not found", "LINK_NOT_FOUND")
		case domain.ErrLinkExpired:
			return utils.Error(c, fiber.StatusGone, "This link has expired", "LINK_EXPIRED")
		case domain.ErrClickLimitReached:
			return utils.Error(c, fiber.StatusGone, "This link has reached its click limit", "CLICK_LIMIT_REACHED")
		case domain.ErrPasswordRequired:
			return utils.Error(c, fiber.StatusUnauthorized, "Password required for this link", "PASSWORD_REQUIRED")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, "Redirect failed", "REDIRECT_FAILED")
		}
	}

	// Fire async analytics tracking — does NOT block redirect
	go h.trackClickAsync(link, clientIP, userAgent, referrer)

	return c.Redirect(targetURL, fiber.StatusFound)
}

func (h *RedirectHandler) Unlock(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")

	var req unlockRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "Invalid request body", "INVALID_BODY")
	}
	if req.Password == "" {
		return utils.Error(c, fiber.StatusBadRequest, "Password is required", "MISSING_PASSWORD")
	}

	originalURL, err := h.linkUseCase.UnlockLink(c.Context(), shortCode, req.Password)
	if err != nil {
		switch err {
		case domain.ErrLinkNotFound:
			return utils.Error(c, fiber.StatusNotFound, "Link not found", "LINK_NOT_FOUND")
		case domain.ErrInvalidPassword:
			return utils.Error(c, fiber.StatusUnauthorized, "Incorrect password", "WRONG_PASSWORD")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, "Unlock failed", "UNLOCK_FAILED")
		}
	}

	return utils.Success(c, fiber.StatusOK, "Link unlocked", fiber.Map{
		"unlocked":     true,
		"original_url": originalURL,
	})
}

// trackClickAsync records an analytics event in the background.
func (h *RedirectHandler) trackClickAsync(link *domain.Link, clientIP, userAgent, referrer string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	country, city, _ := utils.GetLocationFromIP(clientIP)

	event := &domain.AnalyticsEvent{
		LinkID:       link.ID,
		ShortCode:    link.ShortCode,
		ClickedAt:    time.Now(),
		IPAddress:    hashIP(clientIP),
		Country:      country,
		City:         city,
		DeviceType:   parseDeviceType(userAgent),
		Browser:      parseBrowser(userAgent),
		OS:           parseOS(userAgent),
		Referrer:     referrer,
		ReferrerType: classifyReferrer(referrer),
		IsUnique:     false, // Simplified — Redis dedup needed for real unique tracking
		UserAgent:    "",    // Not stored per privacy rules
	}

	_ = h.analyticsUseCase.TrackClick(ctx, event)
}

func hashIP(ip string) string {
	h := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(h[:])
}

func parseDeviceType(ua string) string {
	ua = strings.ToLower(ua)
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		return "mobile"
	}
	if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		return "tablet"
	}
	return "desktop"
}

func parseBrowser(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "edg"):
		return "Edge"
	case strings.Contains(ua, "chrome"):
		return "Chrome"
	case strings.Contains(ua, "firefox"):
		return "Firefox"
	case strings.Contains(ua, "safari"):
		return "Safari"
	case strings.Contains(ua, "opera"):
		return "Opera"
	default:
		return "Other"
	}
}

func parseOS(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "mac os"):
		return "macOS"
	case strings.Contains(ua, "linux"):
		return "Linux"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		return "iOS"
	default:
		return "Other"
	}
}

func classifyReferrer(referrer string) string {
	if referrer == "" {
		return "direct"
	}
	referrer = strings.ToLower(referrer)
	socialDomains := []string{"facebook", "twitter", "instagram", "linkedin", "tiktok", "youtube", "pinterest"}
	searchDomains := []string{"google", "bing", "yahoo", "duckduckgo", "baidu"}
	for _, s := range socialDomains {
		if strings.Contains(referrer, s) {
			return "social"
		}
	}
	for _, s := range searchDomains {
		if strings.Contains(referrer, s) {
			return "search"
		}
	}
	if strings.Contains(referrer, "mailto:") {
		return "email"
	}
	return "other"
}
