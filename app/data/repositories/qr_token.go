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
	col *mongo.Collection
}

type IQrToken interface {
	CreateQrToken(qrToken entities.QrToken) (entities.QrToken, error)
	GetQrTokens(clientId string, branchId string) ([]entities.QrToken, error)
	GetQrTokenById(id primitive.ObjectID) (entities.QrToken, error)
	GetActiveByToken(token string) (entities.QrToken, error)
	Revoke(id primitive.ObjectID, revokedAt time.Time) error
}

func NewQrTokenEntity(resource *db.Resource) IQrToken {
	return &qrTokenEntity{col: resource.AlertDb.Collection("qr_tokens")}
}

func (entity *qrTokenEntity) CreateQrToken(qrToken entities.QrToken) (entities.QrToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	qrToken.Id = primitive.NewObjectID()
	qrToken.CreatedAt = time.Now()
	qrToken.Active = true
	_, err := entity.col.InsertOne(ctx, qrToken)
	return qrToken, err
}

func (entity *qrTokenEntity) GetQrTokens(clientId string, branchId string) ([]entities.QrToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	filter := bson.M{"clientId": clientId}
	if branchId != "" {
		filter["branchId"] = branchId
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := entity.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	var tokens []entities.QrToken
	if err = cursor.All(ctx, &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (entity *qrTokenEntity) GetQrTokenById(id primitive.ObjectID) (entities.QrToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var qrToken entities.QrToken
	err := entity.col.FindOne(ctx, bson.M{"_id": id}).Decode(&qrToken)
	return qrToken, err
}

func (entity *qrTokenEntity) GetActiveByToken(token string) (entities.QrToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var qrToken entities.QrToken
	err := entity.col.FindOne(ctx, bson.M{"token": token, "active": true}).Decode(&qrToken)
	return qrToken, err
}

func (entity *qrTokenEntity) Revoke(id primitive.ObjectID, revokedAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := entity.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"active":    false,
		"revokedAt": revokedAt,
	}})
	return err
}
