package repositories

import (
	"context"
	"time"

	"alert/app/data/entities"
	"alert/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type staffPermissionEntity struct {
	mongo *db.Manager
}

type IStaffPermission interface {
	GetByUserId(clientId string, userId string) (entities.StaffPermission, error)
	GetPermissions(clientId string) ([]entities.StaffPermission, error)
	GetTestRecipients(clientId string, branchId string) ([]entities.StaffPermission, error)
	UpsertPermission(permission entities.StaffPermission) error
}

func NewStaffPermissionEntity(resource *db.Resource) IStaffPermission {
	return &staffPermissionEntity{mongo: resource.Mongo}
}

func (entity *staffPermissionEntity) collection(clientId string) (*mongo.Collection, error) {
	return entity.mongo.CollectionFor(clientId, db.CollectionStaffPermissions)
}

func (entity *staffPermissionEntity) GetByUserId(clientId string, userId string) (entities.StaffPermission, error) {
	var permission entities.StaffPermission
	col, err := entity.collection(clientId)
	if err != nil {
		return permission, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = col.FindOne(ctx, bson.M{"clientId": clientId, "userId": userId}).Decode(&permission)
	return permission, err
}

func (entity *staffPermissionEntity) GetPermissions(clientId string) ([]entities.StaffPermission, error) {
	col, err := entity.collection(clientId)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Find().SetSort(bson.D{{Key: "userId", Value: 1}})
	cursor, err := col.Find(ctx, bson.M{"clientId": clientId}, opts)
	if err != nil {
		return nil, err
	}
	var permissions []entities.StaffPermission
	if err = cursor.All(ctx, &permissions); err != nil {
		return nil, err
	}
	return permissions, nil
}

func (entity *staffPermissionEntity) GetTestRecipients(clientId string, branchId string) ([]entities.StaffPermission, error) {
	col, err := entity.collection(clientId)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	filter := bson.M{
		"clientId":        clientId,
		"branchId":        branchId,
		"isTestRecipient": true,
		"active":          true,
		"phone":           bson.M{"$ne": ""},
	}
	cursor, err := col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	var recipients []entities.StaffPermission
	if err = cursor.All(ctx, &recipients); err != nil {
		return nil, err
	}
	return recipients, nil
}

func (entity *staffPermissionEntity) UpsertPermission(permission entities.StaffPermission) error {
	col, err := entity.collection(permission.ClientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Update().SetUpsert(true)
	_, err = col.UpdateOne(ctx,
		bson.M{"clientId": permission.ClientId, "userId": permission.UserId},
		bson.M{"$set": bson.M{
			"branchId":          permission.BranchId,
			"phone":             permission.Phone,
			"allowedEventTypes": permission.AllowedEventTypes,
			"isTestRecipient":   permission.IsTestRecipient,
			"active":            permission.Active,
			"updatedBy":         permission.UpdatedBy,
			"updatedAt":         time.Now(),
		}}, opts)
	return err
}
