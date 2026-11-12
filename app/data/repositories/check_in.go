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
	// FindCheckInByRef is Client-scoped: for the pre-Confirmation customer paths only.
	FindCheckInByRef(clientId string, id primitive.ObjectID) (entities.CheckIn, error)
	GetCheckInBySessionTokenHash(clientId string, tokenHash string) (entities.CheckIn, error)
	GetCheckInById(scope Scope, id primitive.ObjectID) (entities.CheckIn, error)
	GetActiveCheckIns(scope Scope, now time.Time) ([]entities.CheckIn, error)
	MarkOtpVerified(scope Scope, id primitive.ObjectID, verifiedAt time.Time, sessionTokenHash string, expiresAt time.Time) error
	Checkout(scope Scope, id primitive.ObjectID, checkedOutAt time.Time, checkedOutBy string) error
	SetPushSubscription(scope Scope, id primitive.ObjectID, subscription *entities.PushSubscription) error
	ClearPushSubscription(scope Scope, id primitive.ObjectID) error
	DeleteCheckIn(scope Scope, id primitive.ObjectID) error
	CountActive(scope Scope, now time.Time) (int64, error)
}

func NewCheckInEntity(resource *db.Resource) ICheckIn {
	return &checkInEntity{mongo: resource.Mongo}
}

func (entity *checkInEntity) collection(clientId string) (*mongo.Collection, error) {
	return entity.mongo.CollectionFor(clientId, db.CollectionCheckIns)
}

func activeFilter(scope Scope, now time.Time) bson.M {
	return scope.filter(bson.M{
		"clientId":      scope.ClientId,
		"otpVerifiedAt": bson.M{"$ne": nil},
		"checkedOutAt":  nil,
		"expiresAt":     bson.M{"$gt": now},
	})
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

func (entity *checkInEntity) FindCheckInByRef(clientId string, id primitive.ObjectID) (entities.CheckIn, error) {
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

// GetCheckInById returns the Check-in with that id in that Branch;
// mongo.ErrNoDocuments when it is not in that Branch.
func (entity *checkInEntity) GetCheckInById(scope Scope, id primitive.ObjectID) (entities.CheckIn, error) {
	var checkIn entities.CheckIn
	col, err := entity.collection(scope.ClientId)
	if err != nil {
		return checkIn, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = col.FindOne(ctx, scope.filter(bson.M{"_id": id})).Decode(&checkIn)
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

func (entity *checkInEntity) GetActiveCheckIns(scope Scope, now time.Time) ([]entities.CheckIn, error) {
	col, err := entity.collection(scope.ClientId)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Find().SetSort(bson.D{{Key: "checkedInAt", Value: -1}})
	cursor, err := col.Find(ctx, activeFilter(scope, now), opts)
	if err != nil {
		return nil, err
	}
	checkIns := []entities.CheckIn{}
	if err = cursor.All(ctx, &checkIns); err != nil {
		return nil, err
	}
	return checkIns, nil
}

func (entity *checkInEntity) MarkOtpVerified(scope Scope, id primitive.ObjectID, verifiedAt time.Time, sessionTokenHash string, expiresAt time.Time) error {
	col, err := entity.collection(scope.ClientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.UpdateOne(ctx, scope.filter(bson.M{"_id": id}), bson.M{"$set": bson.M{
		"otpVerifiedAt":    verifiedAt,
		"sessionTokenHash": sessionTokenHash,
		"expiresAt":        expiresAt,
	}})
	return err
}

// Checkout marks the active Check-in with that id in that Branch as checked
// out. It returns mongo.ErrNoDocuments when no such active Check-in exists:
// the id belongs to another Branch, or it is already checked out.
func (entity *checkInEntity) Checkout(scope Scope, id primitive.ObjectID, checkedOutAt time.Time, checkedOutBy string) error {
	col, err := entity.collection(scope.ClientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := col.UpdateOne(ctx, scope.filter(bson.M{"_id": id, "checkedOutAt": nil}), bson.M{"$set": bson.M{
		"checkedOutAt": checkedOutAt,
		"checkedOutBy": checkedOutBy,
	}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (entity *checkInEntity) SetPushSubscription(scope Scope, id primitive.ObjectID, subscription *entities.PushSubscription) error {
	col, err := entity.collection(scope.ClientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.UpdateOne(ctx, scope.filter(bson.M{"_id": id}), bson.M{"$set": bson.M{"pushSubscription": subscription}})
	return err
}

func (entity *checkInEntity) ClearPushSubscription(scope Scope, id primitive.ObjectID) error {
	return entity.SetPushSubscription(scope, id, nil)
}

func (entity *checkInEntity) DeleteCheckIn(scope Scope, id primitive.ObjectID) error {
	col, err := entity.collection(scope.ClientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.DeleteOne(ctx, scope.filter(bson.M{"_id": id}))
	return err
}

func (entity *checkInEntity) CountActive(scope Scope, now time.Time) (int64, error) {
	col, err := entity.collection(scope.ClientId)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return col.CountDocuments(ctx, activeFilter(scope, now))
}
