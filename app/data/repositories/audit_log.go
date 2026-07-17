package repositories

import (
	"context"
	"time"

	"alert/app/data/entities"
	"alert/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type auditLogEntity struct {
	mongo *db.Manager
}

type AuditQuery struct {
	ClientId string
	BranchId string
	Action   string
	From     *time.Time
	To       *time.Time
	Page     int64
	Limit    int64
}

type IAuditLog interface {
	Record(log entities.AuditLog)
	QueryLogs(query AuditQuery) ([]entities.AuditLog, int64, error)
}

func NewAuditLogEntity(resource *db.Resource) IAuditLog {
	return &auditLogEntity{mongo: resource.Mongo}
}

func (entity *auditLogEntity) collection(clientId string) (*mongo.Collection, error) {
	return entity.mongo.CollectionFor(clientId, db.CollectionAuditLogs)
}

func (entity *auditLogEntity) Record(log entities.AuditLog) {
	col, err := entity.collection(log.ClientId)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	log.Id = primitive.NewObjectID()
	if log.OccurredAt.IsZero() {
		log.OccurredAt = time.Now()
	}
	_, _ = col.InsertOne(ctx, log)
}

func (entity *auditLogEntity) QueryLogs(query AuditQuery) ([]entities.AuditLog, int64, error) {
	col, err := entity.collection(query.ClientId)
	if err != nil {
		return nil, 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	filter := bson.M{"clientId": query.ClientId}
	if query.BranchId != "" {
		filter["branchId"] = query.BranchId
	}
	if query.Action != "" {
		filter["action"] = query.Action
	}
	occurred := bson.M{}
	if query.From != nil {
		occurred["$gte"] = *query.From
	}
	if query.To != nil {
		occurred["$lte"] = *query.To
	}
	if len(occurred) > 0 {
		filter["occurredAt"] = occurred
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
		SetSort(bson.D{{Key: "occurredAt", Value: -1}}).
		SetSkip((page - 1) * limit).
		SetLimit(limit)
	cursor, err := col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	logs := []entities.AuditLog{}
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
