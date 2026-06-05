package domain

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnalyticsEvent struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	LinkID       primitive.ObjectID `bson:"linkId" json:"link_id"`
	ShortCode    string             `bson:"shortCode" json:"short_code"`
	ClickedAt    time.Time          `bson:"clickedAt" json:"clicked_at"`
	IPAddress    string             `bson:"ipAddress" json:"ip_address"` // Hashed
	Country      string             `bson:"country" json:"country"`
	City         string             `bson:"city" json:"city"`
	DeviceType   string             `bson:"deviceType" json:"device_type"` // "mobile" | "desktop" | "tablet"
	Browser      string             `bson:"browser" json:"browser"`
	OS           string             `bson:"os" json:"os"`
	Referrer     string             `bson:"referrer" json:"referrer"`
	ReferrerType string             `bson:"referrerType" json:"referrer_type"` // "social" | "email" | "direct" | "search" | "other"
	IsUnique     bool               `bson:"isUnique" json:"is_unique"`
	UserAgent    string             `bson:"userAgent" json:"user_agent"`
}

type ClickStats struct {
	TotalClicks  int64 `json:"total_clicks"`
	UniqueClicks int64 `json:"unique_clicks"`
}

type GeoBreakdown struct {
	Country string `json:"country"`
	City    string `json:"city"`
	Clicks  int64  `json:"clicks"`
}

type DeviceBreakdown struct {
	DeviceType string `json:"device_type"`
	Browser    string `json:"browser"`
	OS         string `json:"os"`
	Clicks     int64  `json:"clicks"`
}

type ReferrerBreakdown struct {
	Referrer     string `json:"referrer"`
	ReferrerType string `json:"referrer_type"`
	Clicks       int64  `json:"clicks"`
}

type TimeSeriesData struct {
	Time   time.Time `json:"time"`
	Clicks int64     `json:"clicks"`
}

type LinkAnalytics struct {
	Stats     ClickStats          `json:"stats"`
	Geo       []GeoBreakdown      `json:"geo"`
	Devices   []DeviceBreakdown   `json:"devices"`
	Referrers []ReferrerBreakdown `json:"referrers"`
	History   []TimeSeriesData    `json:"history"`
}

type AnalyticsRepository interface {
	Insert(ctx context.Context, event *AnalyticsEvent) error
	GetStats(ctx context.Context, linkID primitive.ObjectID) (*ClickStats, error)
	GetGeoAnalytics(ctx context.Context, linkID primitive.ObjectID) ([]GeoBreakdown, error)
	GetDeviceAnalytics(ctx context.Context, linkID primitive.ObjectID) ([]DeviceBreakdown, error)
	GetReferrerAnalytics(ctx context.Context, linkID primitive.ObjectID) ([]ReferrerBreakdown, error)
	GetTimeSeriesAnalytics(ctx context.Context, linkID primitive.ObjectID, start, end time.Time, interval string) ([]TimeSeriesData, error)
}

type AnalyticsUseCase interface {
	TrackClick(ctx context.Context, event *AnalyticsEvent) error
	GetAnalytics(ctx context.Context, linkID primitive.ObjectID) (*LinkAnalytics, error)
	GetGeoAnalytics(ctx context.Context, linkID primitive.ObjectID) ([]GeoBreakdown, error)
	GetDeviceAnalytics(ctx context.Context, linkID primitive.ObjectID) ([]DeviceBreakdown, error)
	GetReferrerAnalytics(ctx context.Context, linkID primitive.ObjectID) ([]ReferrerBreakdown, error)
	GetTimeSeriesAnalytics(ctx context.Context, linkID primitive.ObjectID, start, end time.Time, interval string) ([]TimeSeriesData, error)
}
