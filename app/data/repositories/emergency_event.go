package repositories

import (
	"context"
	"time"

	"alert/app/core/constant"
	"alert/app/data/entities"
	"alert/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type emergencyEventEntity struct {
	mongo *db.Manager
}

type EventQuery struct {
	ClientId  string
	BranchId  string
	EventType string
	From      *time.Time
	To        *time.Time
	Page      int64
	Limit     int64
}

type IEmergencyEvent interface {
	CreateEvent(event entities.EmergencyEvent) (entities.EmergencyEvent, error)
	GetEventById(clientId string, id primitive.ObjectID) (entities.EmergencyEvent, error)
	GetLatestEvent(clientId string, branchId string, eventType string) (*entities.EmergencyEvent, error)
	GetLatestOpenEvent(clientId string, branchId string) (*entities.EmergencyEvent, error)
	QueryEvents(query EventQuery) ([]entities.EmergencyEvent, int64, error)
	UpdateChannelSummary(clientId string, id primitive.ObjectID, summary entities.ChannelSummary, providerReference string) error
	CloseOpenEvents(clientId string, branchId string, closedAt time.Time) error
}

func NewEmergencyEventEntity(resource *db.Resource) IEmergencyEvent {
	return &emergencyEventEntity{mongo: resource.Mongo}
}

func (entity *emergencyEventEntity) collection(clientId string) (*mongo.Collection, error) {
	return entity.mongo.CollectionFor(clientId, db.CollectionEmergencyEvents)
}

func (entity *emergencyEventEntity) CreateEvent(event entities.EmergencyEvent) (entities.EmergencyEvent, error) {
	col, err := entity.collection(event.ClientId)
	if err != nil {
		return event, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	event.Id = primitive.NewObjectID()
	_, err = col.InsertOne(ctx, event)
	return event, err
}

func (entity *emergencyEventEntity) GetEventById(clientId string, id primitive.ObjectID) (entities.EmergencyEvent, error) {
	var event entities.EmergencyEvent
	col, err := entity.collection(clientId)
	if err != nil {
		return event, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = col.FindOne(ctx, bson.M{"_id": id}).Decode(&event)
	return event, err
}

func (entity *emergencyEventEntity) GetLatestEvent(clientId string, branchId string, eventType string) (*entities.EmergencyEvent, error) {
	col, err := entity.collection(clientId)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var event entities.EmergencyEvent
	opts := options.FindOne().SetSort(bson.D{{Key: "sentAt", Value: -1}})
	err = col.FindOne(ctx, bson.M{"clientId": clientId, "branchId": branchId, "eventType": eventType}, opts).Decode(&event)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (entity *emergencyEventEntity) GetLatestOpenEvent(clientId string, branchId string) (*entities.EmergencyEvent, error) {
	col, err := entity.collection(clientId)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var event entities.EmergencyEvent
	opts := options.FindOne().SetSort(bson.D{{Key: "sentAt", Value: -1}})
	filter := bson.M{
		"clientId":  clientId,
		"branchId":  branchId,
		"status":    constant.EventStatusOpen,
		"eventType": bson.M{"$nin": bson.A{constant.EventAllClear, constant.EventTest}},
	}
	err = col.FindOne(ctx, filter, opts).Decode(&event)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (entity *emergencyEventEntity) QueryEvents(query EventQuery) ([]entities.EmergencyEvent, int64, error) {
	col, err := entity.collection(query.ClientId)
	if err != nil {
		return nil, 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	filter := bson.M{"clientId": query.ClientId, "branchId": query.BranchId}
	if query.EventType != "" {
		filter["eventType"] = query.EventType
	}
	sentAt := bson.M{}
	if query.From != nil {
		sentAt["$gte"] = *query.From
	}
	if query.To != nil {
		sentAt["$lte"] = *query.To
	}
	if len(sentAt) > 0 {
		filter["sentAt"] = sentAt
	}
	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "sentAt", Value: -1}}).
		SetSkip((page - 1) * limit).
		SetLimit(limit)
	cursor, err := col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	var events []entities.EmergencyEvent
	if err = cursor.All(ctx, &events); err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (entity *emergencyEventEntity) UpdateChannelSummary(clientId string, id primitive.ObjectID, summary entities.ChannelSummary, providerReference string) error {
	col, err := entity.collection(clientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"channelSummary":    summary,
		"providerReference": providerReference,
	}})
	return err
}

func (entity *emergencyEventEntity) CloseOpenEvents(clientId string, branchId string, closedAt time.Time) error {
	col, err := entity.collection(clientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.UpdateMany(ctx,
		bson.M{"clientId": clientId, "branchId": branchId, "status": constant.EventStatusOpen},
		bson.M{"$set": bson.M{"status": constant.EventStatusClosed, "closedAt": closedAt}})
	return err
}
