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

type qrTokenEntity struct {
	mongo *db.Manager
}

type IQrToken interface {
	CreateQrToken(qrToken entities.QrToken) (entities.QrToken, error)
	GetQrTokens(clientId string, branchId string) ([]entities.QrToken, error)
	GetQrTokenById(clientId string, id primitive.ObjectID) (entities.QrToken, error)
	GetActiveByToken(clientId string, token string) (entities.QrToken, error)
	Revoke(clientId string, id primitive.ObjectID, revokedAt time.Time) error
}

func NewQrTokenEntity(resource *db.Resource) IQrToken {
	return &qrTokenEntity{mongo: resource.Mongo}
}

func (entity *qrTokenEntity) collection(clientId string) (*mongo.Collection, error) {
	return entity.mongo.CollectionFor(clientId, db.CollectionQrTokens)
}

func (entity *qrTokenEntity) CreateQrToken(qrToken entities.QrToken) (entities.QrToken, error) {
	col, err := entity.collection(qrToken.ClientId)
	if err != nil {
		return qrToken, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	qrToken.Id = primitive.NewObjectID()
	qrToken.CreatedAt = time.Now()
	qrToken.Active = true
	_, err = col.InsertOne(ctx, qrToken)
	return qrToken, err
}

func (entity *qrTokenEntity) GetQrTokens(clientId string, branchId string) ([]entities.QrToken, error) {
	col, err := entity.collection(clientId)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	filter := bson.M{"clientId": clientId}
	if branchId != "" {
		filter["branchId"] = branchId
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	var tokens []entities.QrToken
	if err = cursor.All(ctx, &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (entity *qrTokenEntity) GetQrTokenById(clientId string, id primitive.ObjectID) (entities.QrToken, error) {
	var qrToken entities.QrToken
	col, err := entity.collection(clientId)
	if err != nil {
		return qrToken, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = col.FindOne(ctx, bson.M{"_id": id}).Decode(&qrToken)
	return qrToken, err
}

func (entity *qrTokenEntity) GetActiveByToken(clientId string, token string) (entities.QrToken, error) {
	var qrToken entities.QrToken
	col, err := entity.collection(clientId)
	if err != nil {
		return qrToken, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = col.FindOne(ctx, bson.M{"token": token, "active": true}).Decode(&qrToken)
	return qrToken, err
}

func (entity *qrTokenEntity) Revoke(clientId string, id primitive.ObjectID, revokedAt time.Time) error {
	col, err := entity.collection(clientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"active":    false,
		"revokedAt": revokedAt,
	}})
	return err
}
