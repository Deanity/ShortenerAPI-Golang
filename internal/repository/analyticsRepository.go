package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"shortenerapi/internal/domain"
)

type analyticsRepository struct {
	db *mongo.Database
}

func NewAnalyticsRepository(db *mongo.Database) domain.AnalyticsRepository {
	return &analyticsRepository{
		db: db,
	}
}

func (r *analyticsRepository) col() *mongo.Collection {
	return r.db.Collection("analytics_events")
}

func (r *analyticsRepository) Insert(ctx context.Context, event *domain.AnalyticsEvent) error {
	event.ID = primitive.NewObjectID()
	if event.ClickedAt.IsZero() {
		event.ClickedAt = time.Now()
	}
	_, err := r.col().InsertOne(ctx, event)
	return err
}

func (r *analyticsRepository) GetStats(ctx context.Context, linkID primitive.ObjectID) (*domain.ClickStats, error) {
	filter := bson.M{"linkId": linkID}

	total, err := r.col().CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	unique, err := r.col().CountDocuments(ctx, bson.M{"linkId": linkID, "isUnique": true})
	if err != nil {
		return nil, err
	}

	return &domain.ClickStats{
		TotalClicks:  total,
		UniqueClicks: unique,
	}, nil
}

func (r *analyticsRepository) GetGeoAnalytics(ctx context.Context, linkID primitive.ObjectID) ([]domain.GeoBreakdown, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"linkId": linkID}}},
		{{Key: "$group", Value: bson.M{
			"_id":    bson.M{"country": "$country", "city": "$city"},
			"clicks": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{
			"country": "$_id.country",
			"city":    "$_id.city",
			"clicks":  1,
			"_id":     0,
		}}},
		{{Key: "$sort", Value: bson.M{"clicks": -1}}},
	}

	cursor, err := r.col().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []domain.GeoBreakdown
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *analyticsRepository) GetDeviceAnalytics(ctx context.Context, linkID primitive.ObjectID) ([]domain.DeviceBreakdown, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"linkId": linkID}}},
		{{Key: "$group", Value: bson.M{
			"_id":    bson.M{"deviceType": "$deviceType", "browser": "$browser", "os": "$os"},
			"clicks": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{
			"device_type": "$_id.deviceType",
			"browser":     "$_id.browser",
			"os":          "$_id.os",
			"clicks":      1,
			"_id":         0,
		}}},
		{{Key: "$sort", Value: bson.M{"clicks": -1}}},
	}

	cursor, err := r.col().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []domain.DeviceBreakdown
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *analyticsRepository) GetReferrerAnalytics(ctx context.Context, linkID primitive.ObjectID) ([]domain.ReferrerBreakdown, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"linkId": linkID}}},
		{{Key: "$group", Value: bson.M{
			"_id":    bson.M{"referrer": "$referrer", "referrerType": "$referrerType"},
			"clicks": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{
			"referrer":      "$_id.referrer",
			"referrer_type": "$_id.referrerType",
			"clicks":        1,
			"_id":           0,
		}}},
		{{Key: "$sort", Value: bson.M{"clicks": -1}}},
	}

	cursor, err := r.col().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []domain.ReferrerBreakdown
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *analyticsRepository) GetTimeSeriesAnalytics(ctx context.Context, linkID primitive.ObjectID, start, end time.Time, interval string) ([]domain.TimeSeriesData, error) {
	// Build date truncation unit based on interval
	unit := "day"
	switch interval {
	case "hour":
		unit = "hour"
	case "week":
		unit = "week"
	case "month":
		unit = "month"
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"linkId":    linkID,
			"clickedAt": bson.M{"$gte": start, "$lte": end},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"$dateTrunc": bson.M{
					"date": "$clickedAt",
					"unit": unit,
				},
			},
			"clicks": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{
			"time":   "$_id",
			"clicks": 1,
			"_id":    0,
		}}},
		{{Key: "$sort", Value: bson.M{"time": 1}}},
	}

	cursor, err := r.col().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []domain.TimeSeriesData
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
