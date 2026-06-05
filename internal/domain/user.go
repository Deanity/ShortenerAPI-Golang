package domain

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type APIKey struct {
	KeyHash   string    `bson:"keyHash" json:"-"`
	Label     string    `bson:"label" json:"label"`
	Scopes    []string  `bson:"scopes" json:"scopes"`
	TeamID    *string   `bson:"teamId,omitempty" json:"team_id,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"created_at"`
}

type User struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email         string             `bson:"email" json:"email"`
	PasswordHash  string             `bson:"passwordHash" json:"-"`
	Name          string             `bson:"name" json:"name"`
	Plan          string             `bson:"plan" json:"plan"` // "free" | "pro" | "enterprise"
	APIKeys       []APIKey           `bson:"apiKeys" json:"api_keys,omitempty"`
	CustomDomains []string           `bson:"customDomains" json:"custom_domains,omitempty"`
	IsActive      bool               `bson:"isActive" json:"is_active"`
	CreatedAt     time.Time          `bson:"createdAt" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updated_at"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByAPIKeyHash(ctx context.Context, keyHash string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	AddAPIKey(ctx context.Context, userID primitive.ObjectID, apiKey APIKey) error
	DeleteAPIKey(ctx context.Context, userID primitive.ObjectID, keyHash string) error
}

type AuthUseCase interface {
	Register(ctx context.Context, name, email, password string) (*User, error)
	Login(ctx context.Context, email, password string) (string, error) // Returns JWT token
	ValidateToken(ctx context.Context, tokenStr string) (*User, error)
	ValidateAPIKey(ctx context.Context, apiKey string) (*User, *APIKey, error)
	CreateAPIKey(ctx context.Context, userID primitive.ObjectID, label string, scopes []string, teamID *string) (string, error) // Returns plaintext key
	RevokeAPIKey(ctx context.Context, userID primitive.ObjectID, label string) error
	ListAPIKeys(ctx context.Context, userID primitive.ObjectID) ([]APIKey, error)
	AddCustomDomain(ctx context.Context, userID primitive.ObjectID, domain string) error
	DeleteCustomDomain(ctx context.Context, userID primitive.ObjectID, domain string) error
	ListCustomDomains(ctx context.Context, userID primitive.ObjectID) ([]string, error)
}


