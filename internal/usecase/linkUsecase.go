package usecase

import (
	"context"
	"fmt"
	"net/url"
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
	linkRepo         domain.LinkRepository
	analyticsRepo    domain.AnalyticsRepository
	linkCache        *cache.LinkCache
	safeBrowsingKey  string
}

func NewLinkUseCase(
	linkRepo domain.LinkRepository,
	analyticsRepo domain.AnalyticsRepository,
	linkCache *cache.LinkCache,
	safeBrowsingKey string,
) domain.LinkUseCase {
	return &linkUseCase{
		linkRepo:        linkRepo,
		analyticsRepo:   analyticsRepo,
		linkCache:       linkCache,
		safeBrowsingKey: safeBrowsingKey,
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
		return nil, fmt.Errorf("shorten: URL flagged as unsafe by Safe Browsing")
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

	if err := u.linkRepo.Create(ctx, link); err != nil {
		return nil, fmt.Errorf("shorten: create link: %w", err)
	}

	// Warm up Redis cache immediately after creation
	if u.linkCache != nil {
		_ = u.linkCache.SetLink(ctx, shortCode, originalURL)
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
	// Try Redis cache first (fast path)
	if u.linkCache != nil {
		if cached, err := u.linkCache.GetLink(ctx, shortCode); err == nil && cached != "" {
			// We still need the link object for analytics — fetch it, but cache hit means no redirect DB miss
			link, err := u.linkRepo.GetByShortCode(ctx, shortCode)
			if err != nil {
				// Cache hit but DB miss (deleted?) — evict cache and return 404
				_ = u.linkCache.DeleteLink(ctx, shortCode)
				return "", nil, domain.ErrLinkNotFound
			}
			return cached, link, nil
		}
	}

	// Cache miss — fetch from MongoDB
	link, err := u.linkRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return "", nil, domain.ErrLinkNotFound
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

	// Cache for next request
	if u.linkCache != nil {
		_ = u.linkCache.SetLink(ctx, shortCode, link.OriginalURL)
	}

	return link.OriginalURL, link, nil
}

func (u *linkUseCase) UpdateLink(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID, updates map[string]interface{}) (*domain.Link, error) {
	link, err := u.linkRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if link.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	if v, ok := updates["original_url"].(string); ok && v != "" {
		if _, err := url.ParseRequestURI(v); err != nil {
			return nil, fmt.Errorf("updateLink: invalid URL: %w", err)
		}
		link.OriginalURL = v
		// Invalidate cache on URL change
		if u.linkCache != nil {
			_ = u.linkCache.DeleteLink(ctx, link.ShortCode)
		}
	}
	if v, ok := updates["tags"].([]string); ok {
		link.Tags = v
	}
	if v, ok := updates["is_active"].(bool); ok {
		link.IsActive = v
		if !v && u.linkCache != nil {
			_ = u.linkCache.DeleteLink(ctx, link.ShortCode)
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
		_ = u.linkCache.DeleteLink(ctx, link.ShortCode)
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

func (u *linkUseCase) UnlockLink(ctx context.Context, id primitive.ObjectID, password string) (bool, error) {
	link, err := u.linkRepo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	if link.PasswordHash == nil {
		return true, nil
	}
	if !utils.CheckPassword(password, *link.PasswordHash) {
		return false, domain.ErrInvalidPassword
	}
	return true, nil
}
