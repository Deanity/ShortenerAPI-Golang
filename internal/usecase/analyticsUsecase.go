package usecase

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"shortenerapi/internal/domain"
	"shortenerapi/pkg/cache"
)

type analyticsUseCase struct {
	analyticsRepo domain.AnalyticsRepository
	linkRepo      domain.LinkRepository
	linkCache     *cache.LinkCache
}

func NewAnalyticsUseCase(
	analyticsRepo domain.AnalyticsRepository,
	linkRepo domain.LinkRepository,
	linkCache *cache.LinkCache,
) domain.AnalyticsUseCase {
	return &analyticsUseCase{
		analyticsRepo: analyticsRepo,
		linkRepo:      linkRepo,
		linkCache:     linkCache,
	}
}

func (u *analyticsUseCase) TrackClick(ctx context.Context, event *domain.AnalyticsEvent) error {
	// 1. Determine uniqueness using Redis cache
	isUnique := false
	if u.linkCache != nil {
		if unique, err := u.linkCache.IsUniqueVisitor(ctx, event.CustomDomain, event.ShortCode, event.IPAddress); err == nil {

			isUnique = unique
		}
	}
	event.IsUnique = isUnique

	// 2. Increment click counts in MongoDB
	_ = u.linkRepo.IncrementClickCount(ctx, event.ShortCode)
	if isUnique {
		_ = u.linkRepo.IncrementUniqueClickCount(ctx, event.ShortCode)
	}

	// 3. Save the event
	if err := u.analyticsRepo.Insert(ctx, event); err != nil {
		return fmt.Errorf("trackClick: %w", err)
	}
	return nil
}


func (u *analyticsUseCase) GetAnalytics(ctx context.Context, linkID primitive.ObjectID) (*domain.LinkAnalytics, error) {
	stats, err := u.analyticsRepo.GetStats(ctx, linkID)
	if err != nil {
		return nil, fmt.Errorf("getAnalytics: stats: %w", err)
	}
	geo, err := u.analyticsRepo.GetGeoAnalytics(ctx, linkID)
	if err != nil {
		return nil, fmt.Errorf("getAnalytics: geo: %w", err)
	}
	devices, err := u.analyticsRepo.GetDeviceAnalytics(ctx, linkID)
	if err != nil {
		return nil, fmt.Errorf("getAnalytics: devices: %w", err)
	}
	referrers, err := u.analyticsRepo.GetReferrerAnalytics(ctx, linkID)
	if err != nil {
		return nil, fmt.Errorf("getAnalytics: referrers: %w", err)
	}
	return &domain.LinkAnalytics{
		Stats:     *stats,
		Geo:       geo,
		Devices:   devices,
		Referrers: referrers,
		History:   []domain.TimeSeriesData{},
	}, nil
}

func (u *analyticsUseCase) GetGeoAnalytics(ctx context.Context, linkID primitive.ObjectID) ([]domain.GeoBreakdown, error) {
	return u.analyticsRepo.GetGeoAnalytics(ctx, linkID)
}

func (u *analyticsUseCase) GetDeviceAnalytics(ctx context.Context, linkID primitive.ObjectID) ([]domain.DeviceBreakdown, error) {
	return u.analyticsRepo.GetDeviceAnalytics(ctx, linkID)
}

func (u *analyticsUseCase) GetReferrerAnalytics(ctx context.Context, linkID primitive.ObjectID) ([]domain.ReferrerBreakdown, error) {
	return u.analyticsRepo.GetReferrerAnalytics(ctx, linkID)
}

func (u *analyticsUseCase) GetTimeSeriesAnalytics(ctx context.Context, linkID primitive.ObjectID, start, end time.Time, interval string) ([]domain.TimeSeriesData, error) {
	return u.analyticsRepo.GetTimeSeriesAnalytics(ctx, linkID, start, end, interval)
}
