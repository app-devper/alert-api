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
	mongo *db.Manager
}

type ICheckIn interface {
	CreateCheckIn(checkIn entities.CheckIn) (entities.CheckIn, error)
	GetCheckInById(clientId string, id primitive.ObjectID) (entities.CheckIn, error)
	GetCheckInBySessionTokenHash(clientId string, tokenHash string) (entities.CheckIn, error)
	GetActiveCheckIns(clientId string, branchId string, now time.Time) ([]entities.CheckIn, error)
	MarkOtpVerified(clientId string, id primitive.ObjectID, verifiedAt time.Time, sessionTokenHash string, expiresAt time.Time) error
	Checkout(clientId string, id primitive.ObjectID, checkedOutAt time.Time, checkedOutBy string) error
	SetPushSubscription(clientId string, id primitive.ObjectID, subscription *entities.PushSubscription) error
	ClearPushSubscription(clientId string, id primitive.ObjectID) error
	DeleteCheckIn(clientId string, id primitive.ObjectID) error
	CountActive(clientId string, branchId string, now time.Time) (int64, error)
}

func NewCheckInEntity(resource *db.Resource) ICheckIn {
	return &checkInEntity{mongo: resource.Mongo}
}

func (entity *checkInEntity) collection(clientId string) (*mongo.Collection, error) {
	return entity.mongo.CollectionFor(clientId, db.CollectionCheckIns)
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
	col, err := entity.collection(checkIn.ClientId)
	if err != nil {
		return checkIn, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	checkIn.Id = primitive.NewObjectID()
	_, err = col.InsertOne(ctx, checkIn)
	return checkIn, err
}

func (entity *checkInEntity) GetCheckInById(clientId string, id primitive.ObjectID) (entities.CheckIn, error) {
	var checkIn entities.CheckIn
	col, err := entity.collection(clientId)
	if err != nil {
		return checkIn, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = col.FindOne(ctx, bson.M{"_id": id}).Decode(&checkIn)
	return checkIn, err
}

func (entity *checkInEntity) GetCheckInBySessionTokenHash(clientId string, tokenHash string) (entities.CheckIn, error) {
	var checkIn entities.CheckIn
	col, err := entity.collection(clientId)
	if err != nil {
		return checkIn, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = col.FindOne(ctx, bson.M{"clientId": clientId, "sessionTokenHash": tokenHash}).Decode(&checkIn)
	return checkIn, err
}

func (entity *checkInEntity) GetActiveCheckIns(clientId string, branchId string, now time.Time) ([]entities.CheckIn, error) {
	col, err := entity.collection(clientId)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Find().SetSort(bson.D{{Key: "checkedInAt", Value: -1}})
	cursor, err := col.Find(ctx, activeFilter(clientId, branchId, now), opts)
	if err != nil {
		return nil, err
	}
	checkIns := []entities.CheckIn{}
	if err = cursor.All(ctx, &checkIns); err != nil {
		return nil, err
	}
	return checkIns, nil
}

func (entity *checkInEntity) MarkOtpVerified(clientId string, id primitive.ObjectID, verifiedAt time.Time, sessionTokenHash string, expiresAt time.Time) error {
	col, err := entity.collection(clientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"otpVerifiedAt":    verifiedAt,
		"sessionTokenHash": sessionTokenHash,
		"expiresAt":        expiresAt,
	}})
	return err
}

func (entity *checkInEntity) Checkout(clientId string, id primitive.ObjectID, checkedOutAt time.Time, checkedOutBy string) error {
	col, err := entity.collection(clientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.UpdateOne(ctx, bson.M{"_id": id, "checkedOutAt": nil}, bson.M{"$set": bson.M{
		"checkedOutAt": checkedOutAt,
		"checkedOutBy": checkedOutBy,
	}})
	return err
}

func (entity *checkInEntity) SetPushSubscription(clientId string, id primitive.ObjectID, subscription *entities.PushSubscription) error {
	col, err := entity.collection(clientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"pushSubscription": subscription}})
	return err
}

func (entity *checkInEntity) ClearPushSubscription(clientId string, id primitive.ObjectID) error {
	return entity.SetPushSubscription(clientId, id, nil)
}

func (entity *checkInEntity) DeleteCheckIn(clientId string, id primitive.ObjectID) error {
	col, err := entity.collection(clientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (entity *checkInEntity) CountActive(clientId string, branchId string, now time.Time) (int64, error) {
	col, err := entity.collection(clientId)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return col.CountDocuments(ctx, activeFilter(clientId, branchId, now))
}
