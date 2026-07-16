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

type checkInEntity struct {
	col *mongo.Collection
}

type ICheckIn interface {
	CreateCheckIn(checkIn entities.CheckIn) (entities.CheckIn, error)
	GetCheckInById(id primitive.ObjectID) (entities.CheckIn, error)
	GetCheckInBySessionTokenHash(clientId string, tokenHash string) (entities.CheckIn, error)
	GetActiveCheckIns(clientId string, branchId string, now time.Time) ([]entities.CheckIn, error)
	MarkOtpVerified(id primitive.ObjectID, verifiedAt time.Time, sessionTokenHash string, expiresAt time.Time) error
	Checkout(id primitive.ObjectID, checkedOutAt time.Time, checkedOutBy string) error
	SetPushSubscription(id primitive.ObjectID, subscription *entities.PushSubscription) error
	ClearPushSubscription(id primitive.ObjectID) error
	DeleteCheckIn(id primitive.ObjectID) error
	CountActive(clientId string, branchId string, now time.Time) (int64, error)
}

func NewCheckInEntity(resource *db.Resource) ICheckIn {
	return &checkInEntity{col: resource.AlertDb.Collection("check_ins")}
}

func activeFilter(clientId string, branchId string, now time.Time) bson.M {
	return bson.M{
		"clientId":      clientId,
		"branchId":      branchId,
		"otpVerifiedAt": bson.M{"$ne": nil},
		"checkedOutAt":  nil,
		"expiresAt":     bson.M{"$gt": now},
	}
}

func (entity *checkInEntity) CreateCheckIn(checkIn entities.CheckIn) (entities.CheckIn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	checkIn.Id = primitive.NewObjectID()
	_, err := entity.col.InsertOne(ctx, checkIn)
	return checkIn, err
}

func (entity *checkInEntity) GetCheckInById(id primitive.ObjectID) (entities.CheckIn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var checkIn entities.CheckIn
	err := entity.col.FindOne(ctx, bson.M{"_id": id}).Decode(&checkIn)
	return checkIn, err
}

func (entity *checkInEntity) GetCheckInBySessionTokenHash(clientId string, tokenHash string) (entities.CheckIn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var checkIn entities.CheckIn
	err := entity.col.FindOne(ctx, bson.M{"clientId": clientId, "sessionTokenHash": tokenHash}).Decode(&checkIn)
	return checkIn, err
}

func (entity *checkInEntity) GetActiveCheckIns(clientId string, branchId string, now time.Time) ([]entities.CheckIn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Find().SetSort(bson.D{{Key: "checkedInAt", Value: -1}})
	cursor, err := entity.col.Find(ctx, activeFilter(clientId, branchId, now), opts)
	if err != nil {
		return nil, err
	}
	var checkIns []entities.CheckIn
	if err = cursor.All(ctx, &checkIns); err != nil {
		return nil, err
	}
	return checkIns, nil
}

func (entity *checkInEntity) MarkOtpVerified(id primitive.ObjectID, verifiedAt time.Time, sessionTokenHash string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := entity.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"otpVerifiedAt":    verifiedAt,
		"sessionTokenHash": sessionTokenHash,
		"expiresAt":        expiresAt,
	}})
	return err
}

func (entity *checkInEntity) Checkout(id primitive.ObjectID, checkedOutAt time.Time, checkedOutBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := entity.col.UpdateOne(ctx, bson.M{"_id": id, "checkedOutAt": nil}, bson.M{"$set": bson.M{
		"checkedOutAt": checkedOutAt,
		"checkedOutBy": checkedOutBy,
	}})
	return err
}

func (entity *checkInEntity) SetPushSubscription(id primitive.ObjectID, subscription *entities.PushSubscription) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := entity.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"pushSubscription": subscription}})
	return err
}

func (entity *checkInEntity) ClearPushSubscription(id primitive.ObjectID) error {
	return entity.SetPushSubscription(id, nil)
}

func (entity *checkInEntity) DeleteCheckIn(id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := entity.col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (entity *checkInEntity) CountActive(clientId string, branchId string, now time.Time) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return entity.col.CountDocuments(ctx, activeFilter(clientId, branchId, now))
}
