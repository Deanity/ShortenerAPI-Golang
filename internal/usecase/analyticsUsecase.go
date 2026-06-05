package usecase

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"shortenerapi/internal/domain"
)

type analyticsUseCase struct {
	analyticsRepo domain.AnalyticsRepository
}

func NewAnalyticsUseCase(analyticsRepo domain.AnalyticsRepository) domain.AnalyticsUseCase {
	return &analyticsUseCase{
		analyticsRepo: analyticsRepo,
	}
}

func (u *analyticsUseCase) TrackClick(ctx context.Context, event *domain.AnalyticsEvent) error {
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
