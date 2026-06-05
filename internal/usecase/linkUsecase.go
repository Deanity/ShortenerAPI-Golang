package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"shortenerapi/internal/domain"
	"shortenerapi/pkg/cache"
	"shortenerapi/pkg/utils"
)

const (
	DEFAULT_SLUG_LENGTH = 8
	MAX_SLUG_RETRIES    = 5
)

type linkUseCase struct {
	linkRepo        domain.LinkRepository
	analyticsRepo   domain.AnalyticsRepository
	userRepo        domain.UserRepository
	linkCache       *cache.LinkCache
	safeBrowsingKey string
	mainHost        string
}

func NewLinkUseCase(
	linkRepo domain.LinkRepository,
	analyticsRepo domain.AnalyticsRepository,
	userRepo domain.UserRepository,
	linkCache *cache.LinkCache,
	safeBrowsingKey string,
	mainHost string,
) domain.LinkUseCase {
	return &linkUseCase{
		linkRepo:        linkRepo,
		analyticsRepo:   analyticsRepo,
		userRepo:        userRepo,
		linkCache:       linkCache,
		safeBrowsingKey: safeBrowsingKey,
		mainHost:        mainHost,
	}
}


func (u *linkUseCase) Shorten(ctx context.Context, userID primitive.ObjectID, originalURL string, options map[string]interface{}) (*domain.Link, error) {
	// Validate URL format
	if _, err := url.ParseRequestURI(originalURL); err != nil {
		return nil, fmt.Errorf("shorten: invalid URL: %w", err)
	}

	// Malware check via Google Safe Browsing
	safe, err := utils.IsSafeURL(ctx, u.safeBrowsingKey, originalURL)
	if err != nil {
		// Log but don't block on API error
		_ = err
	}
	if !safe {
		return nil, domain.ErrUnsafeURL
	}

	// Determine slug
	var shortCode string
	if slug, ok := options["slug"].(string); ok && slug != "" {
		existing, err := u.linkRepo.GetByShortCode(ctx, slug)
		if err != nil && err != domain.ErrLinkNotFound {
			return nil, fmt.Errorf("shorten: check slug: %w", err)
		}
		if existing != nil {
			return nil, domain.ErrSlugTaken
		}
		shortCode = slug
	} else {
		for i := 0; i < MAX_SLUG_RETRIES; i++ {
			code, err := utils.GenerateSlug(DEFAULT_SLUG_LENGTH)
			if err != nil {
				return nil, fmt.Errorf("shorten: generate slug: %w", err)
			}
			existing, _ := u.linkRepo.GetByShortCode(ctx, code)
			if existing == nil {
				shortCode = code
				break
			}
		}
		if shortCode == "" {
			return nil, fmt.Errorf("shorten: failed to generate unique slug after %d retries", MAX_SLUG_RETRIES)
		}
	}

	link := &domain.Link{
		ShortCode:    shortCode,
		OriginalURL:  originalURL,
		UserID:       userID,
		IsActive:     true,
		ClickCount:   0,
		UniqueClicks: 0,
	}

	// MGMT-2: Custom Domain Validation
	if customDomain, ok := options["custom_domain"].(string); ok && customDomain != "" {
		user, err := u.userRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("shorten: get user: %w", err)
		}
		owned := false
		for _, d := range user.CustomDomains {
			if strings.EqualFold(d, customDomain) {
				owned = true
				break
			}
		}
		if !owned {
			return nil, domain.ErrCustomDomainNotOwned
		}
		link.CustomDomain = &customDomain
	}


	// Apply optional fields
	if tags, ok := options["tags"].([]string); ok {
		link.Tags = tags
	}
	if expiresAt, ok := options["expires_at"].(*time.Time); ok && expiresAt != nil {
		link.ExpiresAt = expiresAt
	}
	if clickLimit, ok := options["click_limit"].(*int); ok && clickLimit != nil {
		link.ClickLimit = clickLimit
	}
	if webhookURL, ok := options["webhook_url"].(string); ok && webhookURL != "" {
		link.WebhookURL = &webhookURL
	}
	// Password protection
	if password, ok := options["password"].(string); ok && password != "" {
		hash, err := utils.HashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("shorten: hash password: %w", err)
		}
		link.PasswordHash = &hash
	}
	// ADV-1: Deep Linking
	if dl, ok := options["deep_link"].(*domain.DeepLink); ok && dl != nil {
		link.DeepLink = dl
	}
	// ADV-2: Geo-Targeting
	if gr, ok := options["geo_rules"].([]domain.GeoRule); ok {
		link.GeoRules = gr
	}
	// ADV-3: Device Targeting
	if dr, ok := options["device_rules"].([]domain.DeviceRule); ok {
		link.DeviceRules = dr
	}
	// ADV-4: Pixel Retargeting
	if px, ok := options["pixels"].([]domain.Pixel); ok {
		link.Pixels = px
	}
	// ADV-5: A/B Testing
	if ab, ok := options["ab_variants"].([]domain.ABVariant); ok {
		link.ABVariants = ab
	}

	if err := u.linkRepo.Create(ctx, link); err != nil {
		return nil, fmt.Errorf("shorten: create link: %w", err)
	}

	// Warm up Redis cache — only for links with no dynamic routing or password/expiry constraints
	if u.linkCache != nil && !hasDynamicRouting(link) &&
		link.PasswordHash == nil && link.ExpiresAt == nil && link.ClickLimit == nil {
		var customDomain string
		if link.CustomDomain != nil {
			customDomain = *link.CustomDomain
		}
		_ = u.linkCache.SetLink(ctx, customDomain, shortCode, originalURL)
	}


	return link, nil
}

func (u *linkUseCase) BulkShorten(ctx context.Context, userID primitive.ObjectID, urls []string, options []map[string]interface{}) ([]*domain.Link, error) {
	results := make([]*domain.Link, 0, len(urls))
	for i, rawURL := range urls {
		var opts map[string]interface{}
		if i < len(options) {
			opts = options[i]
		} else {
			opts = map[string]interface{}{}
		}
		link, err := u.Shorten(ctx, userID, rawURL, opts)
		if err != nil {
			return nil, fmt.Errorf("bulkShorten[%d]: %w", i, err)
		}
		results = append(results, link)
	}
	return results, nil
}

func (u *linkUseCase) GetLink(ctx context.Context, id primitive.ObjectID) (*domain.Link, error) {
	return u.linkRepo.GetByID(ctx, id)
}

func (u *linkUseCase) ResolveRedirect(ctx context.Context, shortCode string, domainName string, clientIP string, userAgent string, referrer string) (string, *domain.Link, error) {
	cleanedDomain := cleanHost(domainName)
	cleanedMainHost := cleanHost(u.mainHost)
	isCustomDomain := cleanedDomain != "" && !strings.EqualFold(cleanedDomain, cleanedMainHost) && !strings.EqualFold(cleanedDomain, "localhost") && !strings.EqualFold(cleanedDomain, "127.0.0.1")

	// Partition domain name for Redis keys
	var cacheDomain string
	if isCustomDomain {
		cacheDomain = cleanedDomain
	}

	// Try Redis cache first (fast path — only for simple links without dynamic routing)
	if u.linkCache != nil {
		if cached, err := u.linkCache.GetLink(ctx, cacheDomain, shortCode); err == nil && cached != "" {
			// Still need link for analytics and validation — fetch from DB
			var link *domain.Link
			var err error
			if isCustomDomain {
				link, err = u.linkRepo.GetByCustomDomainAndCode(ctx, cleanedDomain, shortCode)
			} else {
				link, err = u.linkRepo.GetByShortCode(ctx, shortCode)
			}
			if err != nil {
				// Cache hit but DB miss (deleted?) — evict and return 404
				_ = u.linkCache.DeleteLink(ctx, cacheDomain, shortCode)
				return "", nil, domain.ErrLinkNotFound
			}
			// Perform custom domain restriction
			if link.CustomDomain != nil && *link.CustomDomain != "" {
				if !strings.EqualFold(*link.CustomDomain, cleanedDomain) {
					_ = u.linkCache.DeleteLink(ctx, cacheDomain, shortCode)
					return "", nil, domain.ErrLinkNotFound
				}
			}
			// Perform validations even on cache hits
			if !link.IsActive {
				_ = u.linkCache.DeleteLink(ctx, cacheDomain, shortCode)
				return "", nil, domain.ErrLinkNotFound
			}
			if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
				_ = u.linkCache.DeleteLink(ctx, cacheDomain, shortCode)
				return "", nil, domain.ErrLinkExpired
			}
			if link.ClickLimit != nil && link.ClickCount >= *link.ClickLimit {
				_ = u.linkCache.DeleteLink(ctx, cacheDomain, shortCode)
				return "", nil, domain.ErrClickLimitReached
			}
			if link.PasswordHash != nil {
				return "", nil, domain.ErrPasswordRequired
			}
			targetURL := u.resolveTargetURL(link, clientIP, userAgent)
			return targetURL, link, nil
		}
	}

	// Cache miss — fetch from MongoDB
	var link *domain.Link
	var err error
	if isCustomDomain {
		link, err = u.linkRepo.GetByCustomDomainAndCode(ctx, cleanedDomain, shortCode)
	} else {
		link, err = u.linkRepo.GetByShortCode(ctx, shortCode)
	}
	if err != nil {
		return "", nil, domain.ErrLinkNotFound
	}

	// Perform custom domain restriction
	if link.CustomDomain != nil && *link.CustomDomain != "" {
		if !strings.EqualFold(*link.CustomDomain, cleanedDomain) {
			return "", nil, domain.ErrLinkNotFound
		}
	}

	if !link.IsActive {

		return "", nil, domain.ErrLinkNotFound
	}
	if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
		return "", nil, domain.ErrLinkExpired
	}
	if link.ClickLimit != nil && link.ClickCount >= *link.ClickLimit {
		return "", nil, domain.ErrClickLimitReached
	}
	if link.PasswordHash != nil {
		return "", nil, domain.ErrPasswordRequired
	}

	// Cache for next request — only if no dynamic routing rules
	if u.linkCache != nil && !hasDynamicRouting(link) {
		_ = u.linkCache.SetLink(ctx, cacheDomain, shortCode, link.OriginalURL)
	}

	targetURL := u.resolveTargetURL(link, clientIP, userAgent)
	return targetURL, link, nil
}

func cleanHost(host string) string {
	if idx := strings.Index(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}


// resolveTargetURL applies advanced routing rules in priority order:
// ADV-1 Deep Link → ADV-2 Geo-Targeting → ADV-3 Device Targeting → ADV-5 A/B Testing → original URL.
func (u *linkUseCase) resolveTargetURL(link *domain.Link, clientIP, userAgent string) string {
	// ADV-1: Deep Linking — highest priority (device-native routing)
	if link.DeepLink != nil {
		os := utils.ParseOS(userAgent)
		if os == "iOS" && link.DeepLink.IOS != "" {
			return link.DeepLink.IOS
		}
		if os == "Android" && link.DeepLink.Android != "" {
			return link.DeepLink.Android
		}
	}

	// ADV-2: Geo-Targeting
	if len(link.GeoRules) > 0 {
		country, _, _ := utils.GetLocationFromIP(clientIP)
		if country != "" {
			for _, rule := range link.GeoRules {
				if strings.EqualFold(rule.Country, country) {
					return rule.URL
				}
			}
		}
	}

	// ADV-3: Device Targeting
	if len(link.DeviceRules) > 0 {
		deviceType := utils.ParseDeviceType(userAgent)
		for _, rule := range link.DeviceRules {
			if strings.EqualFold(rule.Device, deviceType) {
				return rule.URL
			}
		}
	}

	// ADV-5: A/B Testing — only if no other rule matched
	if len(link.ABVariants) > 0 {
		return selectABVariant(link.ABVariants)
	}

	return link.OriginalURL
}

// selectABVariant picks a URL from A/B variants using weighted random selection.
func selectABVariant(variants []domain.ABVariant) string {
	total := 0
	for _, v := range variants {
		total += v.Weight
	}
	if total == 0 {
		return variants[0].URL
	}
	r := rand.Intn(total)
	cumulative := 0
	for _, v := range variants {
		cumulative += v.Weight
		if r < cumulative {
			return v.URL
		}
	}
	return variants[len(variants)-1].URL
}

// hasDynamicRouting returns true if the link uses any routing rules that produce
// a different target URL per request (making Redis caching unsafe).
func hasDynamicRouting(link *domain.Link) bool {
	return link.DeepLink != nil ||
		len(link.GeoRules) > 0 ||
		len(link.DeviceRules) > 0 ||
		len(link.ABVariants) > 0
}

func (u *linkUseCase) UpdateLink(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID, updates map[string]interface{}) (*domain.Link, error) {
	link, err := u.linkRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if link.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	var oldDomain string
	if link.CustomDomain != nil {
		oldDomain = *link.CustomDomain
	}

	// MGMT-2: Custom Domain Update & Validation
	if v, ok := updates["custom_domain"].(string); ok {
		if v != "" {
			user, err := u.userRepo.GetByID(ctx, userID)
			if err != nil {
				return nil, fmt.Errorf("updateLink: get user: %w", err)
			}
			owned := false
			for _, d := range user.CustomDomains {
				if strings.EqualFold(d, v) {
					owned = true
					break
				}
			}
			if !owned {
				return nil, domain.ErrCustomDomainNotOwned
			}
			link.CustomDomain = &v
		} else {
			link.CustomDomain = nil
		}
		if u.linkCache != nil {
			_ = u.linkCache.DeleteLink(ctx, oldDomain, link.ShortCode)
		}
	}

	if v, ok := updates["original_url"].(string); ok && v != "" {
		if _, err := url.ParseRequestURI(v); err != nil {
			return nil, fmt.Errorf("updateLink: invalid URL: %w", err)
		}
		link.OriginalURL = v
		// Invalidate cache on URL change
		if u.linkCache != nil {
			_ = u.linkCache.DeleteLink(ctx, oldDomain, link.ShortCode)
		}
	}
	if v, ok := updates["tags"].([]string); ok {
		link.Tags = v
	}
	if v, ok := updates["is_active"].(bool); ok {
		link.IsActive = v
		if !v && u.linkCache != nil {
			_ = u.linkCache.DeleteLink(ctx, oldDomain, link.ShortCode)
		}
	}
	if v, ok := updates["webhook_url"].(string); ok {
		link.WebhookURL = &v
	}
	if v, ok := updates["expires_at"].(*time.Time); ok {
		link.ExpiresAt = v
	}
	if v, ok := updates["click_limit"].(*int); ok {
		link.ClickLimit = v
	}
	// ADV-1: Deep Linking
	if v, ok := updates["deep_link"].(*domain.DeepLink); ok {
		link.DeepLink = v
		// Invalidate cache — link now has dynamic routing
		if u.linkCache != nil {
			_ = u.linkCache.DeleteLink(ctx, oldDomain, link.ShortCode)
		}
	}
	// ADV-2: Geo-Targeting
	if v, ok := updates["geo_rules"].([]domain.GeoRule); ok {
		link.GeoRules = v
		if u.linkCache != nil {
			_ = u.linkCache.DeleteLink(ctx, oldDomain, link.ShortCode)
		}
	}
	// ADV-3: Device Targeting
	if v, ok := updates["device_rules"].([]domain.DeviceRule); ok {
		link.DeviceRules = v
		if u.linkCache != nil {
			_ = u.linkCache.DeleteLink(ctx, oldDomain, link.ShortCode)
		}
	}
	// ADV-4: Pixel Retargeting
	if v, ok := updates["pixels"].([]domain.Pixel); ok {
		link.Pixels = v
	}
	// ADV-5: A/B Testing
	if v, ok := updates["ab_variants"].([]domain.ABVariant); ok {
		link.ABVariants = v
		if u.linkCache != nil {
			_ = u.linkCache.DeleteLink(ctx, oldDomain, link.ShortCode)
		}
	}

	link.UpdatedAt = time.Now()

	if err := u.linkRepo.Update(ctx, link); err != nil {
		return nil, fmt.Errorf("updateLink: %w", err)
	}

	return link, nil
}

func (u *linkUseCase) DeleteLink(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID) error {
	link, err := u.linkRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if link.UserID != userID {
		return domain.ErrUnauthorized
	}
	if u.linkCache != nil {
		var domainName string
		if link.CustomDomain != nil {
			domainName = *link.CustomDomain
		}
		_ = u.linkCache.DeleteLink(ctx, domainName, link.ShortCode)
	}
	return u.linkRepo.Delete(ctx, id)
}


func (u *linkUseCase) ListLinks(ctx context.Context, userID primitive.ObjectID, tag string, isActive *bool, page, perPage int) ([]*domain.Link, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	filter := map[string]interface{}{}
	if tag != "" {
		filter["tags"] = tag
	}
	if isActive != nil {
		filter["isActive"] = *isActive
	}
	return u.linkRepo.List(ctx, userID, filter, page, perPage)
}

func (u *linkUseCase) UnlockLink(ctx context.Context, shortCode string, password string) (string, error) {
	link, err := u.linkRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return "", err
	}
	if link.PasswordHash == nil {
		return link.OriginalURL, nil
	}
	if !utils.CheckPassword(password, *link.PasswordHash) {
		return "", domain.ErrInvalidPassword
	}
	return link.OriginalURL, nil
}
