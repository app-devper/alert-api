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

type messagingConfigEntity struct {
	mongo *db.Manager
}

type IMessagingConfig interface {
	GetConfig(clientId string) (entities.MessagingConfig, error)
	UpsertConfig(config entities.MessagingConfig) error
}

func NewMessagingConfigEntity(resource *db.Resource) IMessagingConfig {
	return &messagingConfigEntity{mongo: resource.Mongo}
}

func (entity *messagingConfigEntity) collection(clientId string) (*mongo.Collection, error) {
	return entity.mongo.CollectionFor(clientId, db.CollectionMessagingConfigs)
}

func (entity *messagingConfigEntity) GetConfig(clientId string) (entities.MessagingConfig, error) {
	var config entities.MessagingConfig
	col, err := entity.collection(clientId)
	if err != nil {
		return config, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = col.FindOne(ctx, bson.M{"clientId": clientId}).Decode(&config)
	if err == mongo.ErrNoDocuments {
		return entities.MessagingConfig{ClientId: clientId}, nil
	}
	return config, err
}

func (entity *messagingConfigEntity) UpsertConfig(config entities.MessagingConfig) error {
	col, err := entity.collection(config.ClientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Update().SetUpsert(true)
	_, err = col.UpdateOne(ctx,
		bson.M{"clientId": config.ClientId},
		bson.M{"$set": bson.M{
			"smsApiUrl":         config.SmsApiUrl,
			"smsBalanceUrl":     config.SmsBalanceUrl,
			"smsApiKey":         config.SmsApiKey,
			"smsApiSecret":      config.SmsApiSecret,
			"smsSenderId":       config.SmsSenderId,
			"smsWebhookSecret":  config.SmsWebhookSecret,
			"lineChannelToken":  config.LineChannelToken,
			"lineChannelSecret": config.LineChannelSecret,
			"updatedBy":         config.UpdatedBy,
			"updatedAt":         time.Now(),
		}}, opts)
	return err
}
