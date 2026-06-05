package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"shortenerapi/internal/domain"
	"shortenerapi/pkg/utils"
)


type authUseCase struct {
	userRepo  domain.UserRepository
	jwtSecret string
}

func NewAuthUseCase(userRepo domain.UserRepository, jwtSecret string) domain.AuthUseCase {
	return &authUseCase{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (u *authUseCase) Register(ctx context.Context, name, email, password string) (*domain.User, error) {
	// Check duplicate email
	existing, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil && err != domain.ErrUserNotFound {
		return nil, fmt.Errorf("register: check email: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("register: hash password: %w", err)
	}

	user := &domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		Plan:         "free",
		IsActive:     true,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("register: create user: %w", err)
	}

	return user, nil
}

func (u *authUseCase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", domain.ErrInvalidCredentials
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		return "", domain.ErrInvalidCredentials
	}

	token, err := u.issueJWT(user)
	if err != nil {
		return "", fmt.Errorf("login: issue token: %w", err)
	}

	return token, nil
}

func (u *authUseCase) ValidateToken(ctx context.Context, tokenStr string) (*domain.User, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrInvalidToken
		}
		return []byte(u.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.ErrInvalidToken
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return nil, domain.ErrInvalidToken
	}

	userID, err := primitive.ObjectIDFromHex(sub)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	return user, nil
}

func (u *authUseCase) ValidateAPIKey(ctx context.Context, apiKey string) (*domain.User, *domain.APIKey, error) {
	hash := hashAPIKey(apiKey)
	user, err := u.userRepo.GetByAPIKeyHash(ctx, hash)
	if err != nil {
		return nil, nil, domain.ErrUnauthorized
	}

	var matchedKey *domain.APIKey
	for _, k := range user.APIKeys {
		if k.KeyHash == hash {
			matchedKey = &k
			break
		}
	}

	return user, matchedKey, nil
}

func (u *authUseCase) CreateAPIKey(ctx context.Context, userID primitive.ObjectID, label string, scopes []string, teamID *string) (string, error) {
	// Generate 32 random bytes → hex string (64 chars)
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("createAPIKey: generate: %w", err)
	}
	plaintext := hex.EncodeToString(raw)
	keyHash := hashAPIKey(plaintext)

	apiKey := domain.APIKey{
		KeyHash:   keyHash,
		Label:     label,
		Scopes:    scopes,
		TeamID:    teamID,
		CreatedAt: time.Now(),
	}

	if err := u.userRepo.AddAPIKey(ctx, userID, apiKey); err != nil {
		return "", fmt.Errorf("createAPIKey: store: %w", err)
	}

	return plaintext, nil
}


func (u *authUseCase) RevokeAPIKey(ctx context.Context, userID primitive.ObjectID, label string) error {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	var keyHash string
	for _, k := range user.APIKeys {
		if k.Label == label {
			keyHash = k.KeyHash
			break
		}
	}
	if keyHash == "" {
		return fmt.Errorf("revokeAPIKey: key with label %q not found", label)
	}

	return u.userRepo.DeleteAPIKey(ctx, userID, keyHash)
}

func (u *authUseCase) ListAPIKeys(ctx context.Context, userID primitive.ObjectID) ([]domain.APIKey, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user.APIKeys, nil
}

// issueJWT creates a signed JWT for the given user (24h expiry).
func (u *authUseCase) issueJWT(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID.Hex(),
		"email": user.Email,
		"plan":  user.Plan,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(u.jwtSecret))
}

// hashAPIKey returns SHA256 hex of the plaintext key.
func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func (u *authUseCase) AddCustomDomain(ctx context.Context, userID primitive.ObjectID, domainName string) error {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Check if already exists
	for _, d := range user.CustomDomains {
		if strings.EqualFold(d, domainName) {
			return nil // already exists, noop
		}
	}

	user.CustomDomains = append(user.CustomDomains, domainName)
	return u.userRepo.Update(ctx, user)
}

func (u *authUseCase) DeleteCustomDomain(ctx context.Context, userID primitive.ObjectID, domainName string) error {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	updatedDomains := make([]string, 0, len(user.CustomDomains))
	for _, d := range user.CustomDomains {
		if !strings.EqualFold(d, domainName) {
			updatedDomains = append(updatedDomains, d)
		}
	}

	user.CustomDomains = updatedDomains
	return u.userRepo.Update(ctx, user)
}

func (u *authUseCase) ListCustomDomains(ctx context.Context, userID primitive.ObjectID) ([]string, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user.CustomDomains, nil
}

