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
	col *mongo.Collection
}

type IOtpRequest interface {
	CreateOtpRequest(request entities.OtpRequest) (entities.OtpRequest, error)
	GetLatestByCheckInId(checkInId primitive.ObjectID) (entities.OtpRequest, error)
	IncrementAttempt(id primitive.ObjectID) (int, error)
	MarkVerified(id primitive.ObjectID, verifiedAt time.Time) error
}

func NewOtpRequestEntity(resource *db.Resource) IOtpRequest {
	return &otpRequestEntity{col: resource.AlertDb.Collection("otp_requests")}
}

func (entity *otpRequestEntity) CreateOtpRequest(request entities.OtpRequest) (entities.OtpRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request.Id = primitive.NewObjectID()
	_, err := entity.col.InsertOne(ctx, request)
	return request, err
}

func (entity *otpRequestEntity) GetLatestByCheckInId(checkInId primitive.ObjectID) (entities.OtpRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var request entities.OtpRequest
	opts := options.FindOne().SetSort(bson.D{{Key: "_id", Value: -1}})
	err := entity.col.FindOne(ctx, bson.M{"checkInId": checkInId}, opts).Decode(&request)
	return request, err
}

func (entity *otpRequestEntity) IncrementAttempt(id primitive.ObjectID) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var updated entities.OtpRequest
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := entity.col.FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{"attemptCount": 1}}, opts).Decode(&updated)
	return updated.AttemptCount, err
}

func (entity *otpRequestEntity) MarkVerified(id primitive.ObjectID, verifiedAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := entity.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"verifiedAt": verifiedAt}})
	return err
}
