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
	col *mongo.Collection
}

type IBranchSetting interface {
	GetSetting(clientId string, branchId string) (entities.BranchSetting, error)
	UpsertSetting(setting entities.BranchSetting) error
	SetPinHash(clientId string, branchId string, pinHash string, confirmMethod string, updatedBy string) error
}

func NewBranchSettingEntity(resource *db.Resource) IBranchSetting {
	return &branchSettingEntity{col: resource.AlertDb.Collection("branch_settings")}
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var setting entities.BranchSetting
	err := entity.col.FindOne(ctx, bson.M{"clientId": clientId, "branchId": branchId}).Decode(&setting)
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Update().SetUpsert(true)
	_, err := entity.col.UpdateOne(ctx,
		bson.M{"clientId": setting.ClientId, "branchId": setting.BranchId},
		bson.M{"$set": bson.M{
			"shopName":           setting.ShopName,
			"retentionHours":     setting.RetentionHours,
			"cooldownSeconds":    setting.CooldownSeconds,
			"confirmMethod":      setting.ConfirmMethod,
			"smsCreditThreshold": setting.SmsCreditThreshold,
			"contactChannel":     setting.ContactChannel,
			"updatedBy":          setting.UpdatedBy,
			"updatedAt":          time.Now(),
		}}, opts)
	return err
}

func (entity *branchSettingEntity) SetPinHash(clientId string, branchId string, pinHash string, confirmMethod string, updatedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Update().SetUpsert(true)
	_, err := entity.col.UpdateOne(ctx,
		bson.M{"clientId": clientId, "branchId": branchId},
		bson.M{"$set": bson.M{
			"pinHash":       pinHash,
			"confirmMethod": confirmMethod,
			"updatedBy":     updatedBy,
			"updatedAt":     time.Now(),
		}}, opts)
	return err
}
