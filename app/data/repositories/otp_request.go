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

type otpRequestEntity struct {
	mongo *db.Manager
}

type IOtpRequest interface {
	CreateOtpRequest(request entities.OtpRequest) (entities.OtpRequest, error)
	GetLatestByCheckInId(clientId string, checkInId primitive.ObjectID) (entities.OtpRequest, error)
	IncrementAttempt(clientId string, id primitive.ObjectID) (int, error)
	MarkVerified(clientId string, id primitive.ObjectID, verifiedAt time.Time) error
}

func NewOtpRequestEntity(resource *db.Resource) IOtpRequest {
	return &otpRequestEntity{mongo: resource.Mongo}
}

func (entity *otpRequestEntity) collection(clientId string) (*mongo.Collection, error) {
	return entity.mongo.CollectionFor(clientId, db.CollectionOtpRequests)
}

func (entity *otpRequestEntity) CreateOtpRequest(request entities.OtpRequest) (entities.OtpRequest, error) {
	col, err := entity.collection(request.ClientId)
	if err != nil {
		return request, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request.Id = primitive.NewObjectID()
	_, err = col.InsertOne(ctx, request)
	return request, err
}

func (entity *otpRequestEntity) GetLatestByCheckInId(clientId string, checkInId primitive.ObjectID) (entities.OtpRequest, error) {
	var request entities.OtpRequest
	col, err := entity.collection(clientId)
	if err != nil {
		return request, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.FindOne().SetSort(bson.D{{Key: "_id", Value: -1}})
	err = col.FindOne(ctx, bson.M{"checkInId": checkInId}, opts).Decode(&request)
	return request, err
}

func (entity *otpRequestEntity) IncrementAttempt(clientId string, id primitive.ObjectID) (int, error) {
	col, err := entity.collection(clientId)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var updated entities.OtpRequest
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err = col.FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"attemptCount": 1}}, opts).Decode(&updated)
	return updated.AttemptCount, err
}

func (entity *otpRequestEntity) MarkVerified(clientId string, id primitive.ObjectID, verifiedAt time.Time) error {
	col, err := entity.collection(clientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"verifiedAt": verifiedAt}})
	return err
}
