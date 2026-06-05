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

type linkRepository struct {
	db *mongo.Database
}

func NewLinkRepository(db *mongo.Database) domain.LinkRepository {
	return &linkRepository{
		db: db,
	}
}

func (r *linkRepository) col() *mongo.Collection {
	return r.db.Collection("links")
}

func (r *linkRepository) Create(ctx context.Context, link *domain.Link) error {
	link.ID = primitive.NewObjectID()
	now := time.Now()
	link.CreatedAt = now
	link.UpdatedAt = now

	_, err := r.col().InsertOne(ctx, link)
	return err
}

func (r *linkRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Link, error) {
	var link domain.Link
	err := r.col().FindOne(ctx, bson.M{"_id": id}).Decode(&link)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrLinkNotFound
	}
	return &link, err
}

func (r *linkRepository) GetByShortCode(ctx context.Context, shortCode string) (*domain.Link, error) {
	var link domain.Link
	err := r.col().FindOne(ctx, bson.M{"shortCode": shortCode}).Decode(&link)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrLinkNotFound
	}
	return &link, err
}

func (r *linkRepository) GetByCustomDomainAndCode(ctx context.Context, domainName, shortCode string) (*domain.Link, error) {
	var link domain.Link
	err := r.col().FindOne(ctx, bson.M{
		"customDomain": domainName,
		"shortCode":    shortCode,
	}).Decode(&link)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrLinkNotFound
	}
	return &link, err
}

func (r *linkRepository) Update(ctx context.Context, link *domain.Link) error {
	link.UpdatedAt = time.Now()
	_, err := r.col().ReplaceOne(ctx, bson.M{"_id": link.ID}, link)
	return err
}

func (r *linkRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col().DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *linkRepository) List(ctx context.Context, userID primitive.ObjectID, filter map[string]interface{}, page, perPage int) ([]*domain.Link, int, error) {
	query := bson.M{"userId": userID}
	for k, v := range filter {
		query[k] = v
	}

	total, err := r.col().CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * perPage)
	limit := int64(perPage)
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.col().Find(ctx, query, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var links []*domain.Link
	if err := cursor.All(ctx, &links); err != nil {
		return nil, 0, err
	}

	return links, int(total), nil
}

func (r *linkRepository) IncrementClickCount(ctx context.Context, shortCode string) error {
	_, err := r.col().UpdateOne(ctx,
		bson.M{"shortCode": shortCode},
		bson.M{"$inc": bson.M{"clickCount": 1}},
	)
	return err
}

func (r *linkRepository) IncrementUniqueClickCount(ctx context.Context, shortCode string) error {
	_, err := r.col().UpdateOne(ctx,
		bson.M{"shortCode": shortCode},
		bson.M{"$inc": bson.M{"uniqueClicks": 1}},
	)
	return err
}
