package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"shortenerapi/internal/domain"
	"shortenerapi/pkg/utils"
)

type RedirectHandler struct {
	linkUseCase       domain.LinkUseCase
	analyticsUseCase  domain.AnalyticsUseCase
	webhookMaxRetries int
	webhookTimeout    time.Duration
}

func NewRedirectHandler(
	linkUseCase domain.LinkUseCase,
	analyticsUseCase domain.AnalyticsUseCase,
	webhookMaxRetries int,
	webhookTimeoutSeconds int,
) *RedirectHandler {
	timeout := time.Duration(webhookTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	retries := webhookMaxRetries
	if retries <= 0 {
		retries = 3
	}
	return &RedirectHandler{
		linkUseCase:       linkUseCase,
		analyticsUseCase:  analyticsUseCase,
		webhookMaxRetries: retries,
		webhookTimeout:    timeout,
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

	targetURL, link, err := h.linkUseCase.ResolveRedirect(c.Context(), shortCode, c.Hostname(), clientIP, userAgent, referrer)
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
	go h.trackClickAsync(targetURL, link, clientIP, userAgent, referrer)


	// ADV-4: Pixel Retargeting — render HTML page instead of bare 302
	if len(link.Pixels) > 0 {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(renderPixelPage(targetURL, link.Pixels))
	}

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

// trackClickAsync records an analytics event in the background and sends webhooks.
func (h *RedirectHandler) trackClickAsync(targetURL string, link *domain.Link, clientIP, userAgent, referrer string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	country, city, _ := utils.GetLocationFromIP(clientIP)

	var customDomain string
	if link.CustomDomain != nil {
		customDomain = *link.CustomDomain
	}

	event := &domain.AnalyticsEvent{
		LinkID:       link.ID,
		ShortCode:    link.ShortCode,
		CustomDomain: customDomain,
		ClickedAt:    time.Now(),
		IPAddress:    hashIP(clientIP),
		Country:      country,
		City:         city,
		DeviceType:   utils.ParseDeviceType(userAgent),
		Browser:      utils.ParseBrowser(userAgent),
		OS:           utils.ParseOS(userAgent),
		Referrer:     referrer,
		ReferrerType: utils.ClassifyReferrer(referrer),
		IsUnique:     false,
		UserAgent:    "", // Not stored per privacy rules
	}

	_ = h.analyticsUseCase.TrackClick(ctx, event)

	// Send real-time webhook async if configured
	if link.WebhookURL != nil && *link.WebhookURL != "" {
		payload := map[string]interface{}{
			"event":         "link.click",
			"link_id":       link.ID.Hex(),
			"short_code":    link.ShortCode,
			"custom_domain": customDomain,
			"original_url":  link.OriginalURL,
			"target_url":    targetURL,
			"clicked_at":    event.ClickedAt.Format(time.RFC3339),
			"ip_address":    event.IPAddress,
			"country":       event.Country,
			"city":          event.City,
			"device_type":   event.DeviceType,
			"browser":       event.Browser,
			"os":            event.OS,
			"referrer":      event.Referrer,
			"referrer_type": event.ReferrerType,
			"is_unique":     event.IsUnique,
		}
		_ = utils.SendWebhook(context.Background(), *link.WebhookURL, payload, h.webhookMaxRetries, h.webhookTimeout)
	}
}


func hashIP(ip string) string {
	h := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(h[:])
}

// renderPixelPage generates an HTML page that fires pixel scripts then redirects.
// This implements ADV-4: Pixel Retargeting (Google Tag / Facebook Pixel).
func renderPixelPage(targetURL string, pixels []domain.Pixel) string {
	var scripts strings.Builder

	for _, p := range pixels {
		switch p.Type {
		case "gtag":
			scripts.WriteString(fmt.Sprintf(`
<!-- Google Tag: %s -->
<script async src="https://www.googletagmanager.com/gtag/js?id=%s"></script>
<script>
window.dataLayer=window.dataLayer||[];
function gtag(){dataLayer.push(arguments);}
gtag('js',new Date());
gtag('config','%s',{'send_page_view':true});
</script>`, p.ID, p.ID, p.ID))

		case "fbpixel":
			scripts.WriteString(fmt.Sprintf(`
<!-- Facebook Pixel: %s -->
<script>
!function(f,b,e,v,n,t,s){if(f.fbq)return;n=f.fbq=function(){n.callMethod?
n.callMethod.apply(n,arguments):n.queue.push(arguments)};if(!f._fbq)f._fbq=n;
n.push=n;n.loaded=!0;n.version='2.0';n.queue=[];t=b.createElement(e);t.async=!0;
t.src=v;s=b.getElementsByTagName(e)[0];s.parentNode.insertBefore(t,s)}(window,
document,'script','https://connect.facebook.net/en_US/fbevents.js');
fbq('init','%s');
fbq('track','PageView');
</script>
<noscript><img height="1" width="1" style="display:none"
src="https://www.facebook.com/tr?id=%s&ev=PageView&noscript=1"/></noscript>`,
				p.ID, p.ID, p.ID))
		}
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="robots" content="noindex,nofollow">
<meta http-equiv="refresh" content="0;url=%s">
%s
<script>
(function(){
  try { window.location.replace(%q); }
  catch(e) { window.location.href = %q; }
})();
</script>
</head>
<body></body>
</html>`, targetURL, scripts.String(), targetURL, targetURL)
}
