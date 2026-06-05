package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"shortenerapi/internal/domain"
)

type userRepository struct {
	db *mongo.Database
}

func NewUserRepository(db *mongo.Database) domain.UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) col() *mongo.Collection {
	return r.db.Collection("users")
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	user.ID = primitive.NewObjectID()
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.col().InsertOne(ctx, user)
	return err
}

func (r *userRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.User, error) {
	var user domain.User
	err := r.col().FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrUserNotFound
	}
	return &user, err
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.col().FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrUserNotFound
	}
	return &user, err
}

func (r *userRepository) GetByAPIKeyHash(ctx context.Context, keyHash string) (*domain.User, error) {
	var user domain.User
	err := r.col().FindOne(ctx, bson.M{"apiKeys.keyHash": keyHash}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrUserNotFound
	}
	return &user, err
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now()
	_, err := r.col().ReplaceOne(ctx, bson.M{"_id": user.ID}, user)
	return err
}

func (r *userRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col().DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *userRepository) AddAPIKey(ctx context.Context, userID primitive.ObjectID, apiKey domain.APIKey) error {
	_, err := r.col().UpdateOne(ctx,
		bson.M{"_id": userID},
		bson.M{
			"$push": bson.M{"apiKeys": apiKey},
			"$set":  bson.M{"updatedAt": time.Now()},
		},
	)
	return err
}

func (r *userRepository) DeleteAPIKey(ctx context.Context, userID primitive.ObjectID, keyHash string) error {
	_, err := r.col().UpdateOne(ctx,
		bson.M{"_id": userID},
		bson.M{
			"$pull": bson.M{"apiKeys": bson.M{"keyHash": keyHash}},
			"$set":  bson.M{"updatedAt": time.Now()},
		},
		options.Update(),
	)
	return err
}
