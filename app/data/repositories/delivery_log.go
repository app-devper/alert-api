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

type deliveryLogEntity struct {
	mongo *db.Manager
}

type IDeliveryLog interface {
	CreateMany(clientId string, logs []entities.DeliveryLog) error
	GetByEventId(clientId string, eventId primitive.ObjectID) ([]entities.DeliveryLog, error)
	GetFailedByEventId(clientId string, eventId primitive.ObjectID) ([]entities.DeliveryLog, error)
	UpdateStatusByProviderReference(clientId string, providerReference string, status string, providerStatus string, at time.Time) (entities.DeliveryLog, error)
	SummarizeByEventId(clientId string, eventId primitive.ObjectID) (entities.ChannelSummary, error)
}

func NewDeliveryLogEntity(resource *db.Resource) IDeliveryLog {
	return &deliveryLogEntity{mongo: resource.Mongo}
}

func (entity *deliveryLogEntity) collection(clientId string) (*mongo.Collection, error) {
	return entity.mongo.CollectionFor(clientId, db.CollectionDeliveryLogs)
}

func (entity *deliveryLogEntity) CreateMany(clientId string, logs []entities.DeliveryLog) error {
	if len(logs) == 0 {
		return nil
	}
	col, err := entity.collection(clientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	docs := make([]interface{}, 0, len(logs))
	for _, logEntry := range logs {
		if logEntry.Id.IsZero() {
			logEntry.Id = primitive.NewObjectID()
		}
		docs = append(docs, logEntry)
	}
	_, err = col.InsertMany(ctx, docs)
	return err
}

func (entity *deliveryLogEntity) GetByEventId(clientId string, eventId primitive.ObjectID) ([]entities.DeliveryLog, error) {
	return entity.findByEvent(clientId, bson.M{"eventId": eventId})
}

func (entity *deliveryLogEntity) GetFailedByEventId(clientId string, eventId primitive.ObjectID) ([]entities.DeliveryLog, error) {
	return entity.findByEvent(clientId, bson.M{"eventId": eventId, "status": constant.DeliveryFailed})
}

func (entity *deliveryLogEntity) findByEvent(clientId string, filter bson.M) ([]entities.DeliveryLog, error) {
	col, err := entity.collection(clientId)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Find().SetSort(bson.D{{Key: "queuedAt", Value: 1}})
	cursor, err := col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	logs := []entities.DeliveryLog{}
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

func (entity *deliveryLogEntity) UpdateStatusByProviderReference(clientId string, providerReference string, status string, providerStatus string, at time.Time) (entities.DeliveryLog, error) {
	var updated entities.DeliveryLog
	col, err := entity.collection(clientId)
	if err != nil {
		return updated, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	set := bson.M{"status": status, "providerStatus": providerStatus}
	if status == constant.DeliveryDelivered {
		set["deliveredAt"] = at
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err = col.FindOneAndUpdate(ctx, bson.M{"providerReference": providerReference}, bson.M{"$set": set}, opts).Decode(&updated)
	return updated, err
}

func (entity *deliveryLogEntity) SummarizeByEventId(clientId string, eventId primitive.ObjectID) (entities.ChannelSummary, error) {
	col, err := entity.collection(clientId)
	if err != nil {
		return entities.ChannelSummary{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"eventId": eventId}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"channel": "$channel", "status": "$status"},
			"count": bson.M{"$sum": 1},
		}}},
	}
	cursor, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return entities.ChannelSummary{}, err
	}
	var rows []struct {
		Id struct {
			Channel string `bson:"channel"`
			Status  string `bson:"status"`
		} `bson:"_id"`
		Count int `bson:"count"`
	}
	if err = cursor.All(ctx, &rows); err != nil {
		return entities.ChannelSummary{}, err
	}
	summary := entities.ChannelSummary{}
	for _, row := range rows {
		stat := statFor(&summary, row.Id.Channel)
		if stat == nil {
			continue
		}
		switch row.Id.Status {
		case constant.DeliverySent, constant.DeliveryQueued:
			stat.Sent += row.Count
		case constant.DeliveryDelivered:
			stat.Sent += row.Count
			stat.Delivered += row.Count
		case constant.DeliveryFailed:
			stat.Failed += row.Count
		}
	}
	return summary, nil
}

func statFor(summary *entities.ChannelSummary, channel string) *entities.ChannelStat {
	switch channel {
	case constant.ChannelSms:
		return &summary.Sms
	case constant.ChannelPush:
		return &summary.Push
	case constant.ChannelLine:
		return &summary.Line
	}
	return nil
}
