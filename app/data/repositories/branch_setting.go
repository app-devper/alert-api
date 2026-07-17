package repositories

import (
	"context"
	"time"

	"alert/app/core/constant"
	"alert/app/data/entities"
	"alert/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type branchSettingEntity struct {
	mongo *db.Manager
}

type IBranchSetting interface {
	GetSetting(clientId string, branchId string) (entities.BranchSetting, error)
	UpsertSetting(setting entities.BranchSetting) error
	SetPinHash(clientId string, branchId string, pinHash string, confirmMethod string, updatedBy string) error
}

func NewBranchSettingEntity(resource *db.Resource) IBranchSetting {
	return &branchSettingEntity{mongo: resource.Mongo}
}

func (entity *branchSettingEntity) collection(clientId string) (*mongo.Collection, error) {
	return entity.mongo.CollectionFor(clientId, db.CollectionBranchSettings)
}

func defaultSetting(clientId string, branchId string) entities.BranchSetting {
	return entities.BranchSetting{
		ClientId:        clientId,
		BranchId:        branchId,
		RetentionHours:  constant.DefaultRetentionHours,
		CooldownSeconds: constant.DefaultCooldownSeconds,
		ConfirmMethod:   constant.ConfirmHold3s,
	}
}

func (entity *branchSettingEntity) GetSetting(clientId string, branchId string) (entities.BranchSetting, error) {
	var setting entities.BranchSetting
	col, err := entity.collection(clientId)
	if err != nil {
		return setting, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = col.FindOne(ctx, bson.M{"clientId": clientId, "branchId": branchId}).Decode(&setting)
	if err == mongo.ErrNoDocuments {
		return defaultSetting(clientId, branchId), nil
	}
	if err != nil {
		return setting, err
	}
	if setting.RetentionHours < constant.MinRetentionHours || setting.RetentionHours > constant.MaxRetentionHours {
		setting.RetentionHours = constant.DefaultRetentionHours
	}
	if setting.ConfirmMethod == "" {
		setting.ConfirmMethod = constant.ConfirmHold3s
	}
	return setting, nil
}

func (entity *branchSettingEntity) UpsertSetting(setting entities.BranchSetting) error {
	col, err := entity.collection(setting.ClientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Update().SetUpsert(true)
	_, err = col.UpdateOne(ctx,
		bson.M{"clientId": setting.ClientId, "branchId": setting.BranchId},
		bson.M{"$set": bson.M{
			"shopName":           setting.ShopName,
			"retentionHours":     setting.RetentionHours,
			"cooldownSeconds":    setting.CooldownSeconds,
			"confirmMethod":      setting.ConfirmMethod,
			"skipOtp":            setting.SkipOtp,
			"smsCreditThreshold": setting.SmsCreditThreshold,
			"contactChannel":     setting.ContactChannel,
			"updatedBy":          setting.UpdatedBy,
			"updatedAt":          time.Now(),
		}}, opts)
	return err
}

func (entity *branchSettingEntity) SetPinHash(clientId string, branchId string, pinHash string, confirmMethod string, updatedBy string) error {
	col, err := entity.collection(clientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Update().SetUpsert(true)
	_, err = col.UpdateOne(ctx,
		bson.M{"clientId": clientId, "branchId": branchId},
		bson.M{"$set": bson.M{
			"pinHash":       pinHash,
			"confirmMethod": confirmMethod,
			"updatedBy":     updatedBy,
			"updatedAt":     time.Now(),
		}}, opts)
	return err
}
