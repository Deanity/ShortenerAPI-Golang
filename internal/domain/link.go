package domain

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DeepLink struct {
	IOS     string `bson:"ios,omitempty" json:"ios,omitempty"`
	Android string `bson:"android,omitempty" json:"android,omitempty"`
}

type GeoRule struct {
	Country string `bson:"country" json:"country"`
	URL     string `bson:"url" json:"url"`
}

type DeviceRule struct {
	Device string `bson:"device" json:"device"`
	URL    string `bson:"url" json:"url"`
}

type ABVariant struct {
	URL    string `bson:"url" json:"url"`
	Weight int    `bson:"weight" json:"weight"`
}

type Pixel struct {
	Type string `bson:"type" json:"type"` // "gtag" | "fbpixel"
	ID   string `bson:"id" json:"id"`
}

type Link struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ShortCode    string             `bson:"shortCode" json:"short_code"`
	OriginalURL  string             `bson:"originalUrl" json:"original_url"`
	CustomSlug   *string            `bson:"customSlug" json:"custom_slug,omitempty"`
	CustomDomain *string            `bson:"customDomain" json:"custom_domain,omitempty"`
	UserID       primitive.ObjectID `bson:"userId" json:"user_id"`
	Tags         []string           `bson:"tags" json:"tags,omitempty"`
	IsActive     bool               `bson:"isActive" json:"is_active"`
	PasswordHash *string            `bson:"passwordHash" json:"-"`
	ExpiresAt    *time.Time         `bson:"expiresAt" json:"expires_at,omitempty"`
	ClickLimit   *int               `bson:"clickLimit" json:"click_limit,omitempty"`
	ClickCount   int                `bson:"clickCount" json:"click_count"`
	UniqueClicks int                `bson:"uniqueClicks" json:"unique_clicks"`
	DeepLink     *DeepLink          `bson:"deepLink" json:"deep_link,omitempty"`
	GeoRules     []GeoRule          `bson:"geoRules" json:"geo_rules,omitempty"`
	DeviceRules  []DeviceRule       `bson:"deviceRules" json:"device_rules,omitempty"`
	ABVariants   []ABVariant        `bson:"abVariants" json:"ab_variants,omitempty"`
	Pixels       []Pixel            `bson:"pixels" json:"pixels,omitempty"`
	WebhookURL   *string            `bson:"webhookUrl" json:"webhook_url,omitempty"`
	CreatedAt    time.Time          `bson:"createdAt" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updated_at"`
}

type LinkRepository interface {
	Create(ctx context.Context, link *Link) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*Link, error)
	GetByShortCode(ctx context.Context, shortCode string) (*Link, error)
	GetByCustomDomainAndCode(ctx context.Context, domain, shortCode string) (*Link, error)
	Update(ctx context.Context, link *Link) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	List(ctx context.Context, userID primitive.ObjectID, filter map[string]interface{}, page, perPage int) ([]*Link, int, error)
	IncrementClickCount(ctx context.Context, shortCode string) error
	IncrementUniqueClickCount(ctx context.Context, shortCode string) error
}

type LinkUseCase interface {
	Shorten(ctx context.Context, userID primitive.ObjectID, originalURL string, options map[string]interface{}) (*Link, error)
	BulkShorten(ctx context.Context, userID primitive.ObjectID, urls []string, options []map[string]interface{}) ([]*Link, error)
	GetLink(ctx context.Context, id primitive.ObjectID) (*Link, error)
	ResolveRedirect(ctx context.Context, shortCode string, domain string, clientIP string, userAgent string, referrer string) (string, *Link, error)
	UpdateLink(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID, updates map[string]interface{}) (*Link, error)
	DeleteLink(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID) error
	ListLinks(ctx context.Context, userID primitive.ObjectID, tag string, isActive *bool, page, perPage int) ([]*Link, int, error)
	UnlockLink(ctx context.Context, shortCode string, password string) (string, error)
}
