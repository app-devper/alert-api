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

type messageTemplateEntity struct {
	mongo *db.Manager
}

type IMessageTemplate interface {
	GetTemplates(clientId string) ([]entities.MessageTemplate, error)
	GetTemplateById(clientId string, id primitive.ObjectID) (entities.MessageTemplate, error)
	GetActiveTemplateByCode(clientId string, code string) (entities.MessageTemplate, error)
	CreateTemplate(template entities.MessageTemplate) (entities.MessageTemplate, error)
	UpdateTemplate(clientId string, id primitive.ObjectID, template entities.MessageTemplate) error
	SetActive(clientId string, id primitive.ObjectID, active bool, updatedBy string) error
	CountActiveByCode(clientId string, code string, excludeId *primitive.ObjectID) (int64, error)
}

func NewMessageTemplateEntity(resource *db.Resource) IMessageTemplate {
	return &messageTemplateEntity{mongo: resource.Mongo}
}

func (entity *messageTemplateEntity) collection(clientId string) (*mongo.Collection, error) {
	return entity.mongo.CollectionFor(clientId, db.CollectionMessageTemplates)
}

func (entity *messageTemplateEntity) GetTemplates(clientId string) ([]entities.MessageTemplate, error) {
	col, err := entity.collection(clientId)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Find().SetSort(bson.D{{Key: "code", Value: 1}})
	cursor, err := col.Find(ctx, bson.M{"clientId": clientId}, opts)
	if err != nil {
		return nil, err
	}
	var templates []entities.MessageTemplate
	if err = cursor.All(ctx, &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

func (entity *messageTemplateEntity) GetTemplateById(clientId string, id primitive.ObjectID) (entities.MessageTemplate, error) {
	var template entities.MessageTemplate
	col, err := entity.collection(clientId)
	if err != nil {
		return template, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = col.FindOne(ctx, bson.M{"_id": id}).Decode(&template)
	return template, err
}

func (entity *messageTemplateEntity) GetActiveTemplateByCode(clientId string, code string) (entities.MessageTemplate, error) {
	var template entities.MessageTemplate
	col, err := entity.collection(clientId)
	if err != nil {
		return template, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = col.FindOne(ctx, bson.M{"clientId": clientId, "code": code, "active": true}).Decode(&template)
	return template, err
}

func (entity *messageTemplateEntity) CreateTemplate(template entities.MessageTemplate) (entities.MessageTemplate, error) {
	col, err := entity.collection(template.ClientId)
	if err != nil {
		return template, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	template.Id = primitive.NewObjectID()
	template.UpdatedAt = time.Now()
	_, err = col.InsertOne(ctx, template)
	return template, err
}

func (entity *messageTemplateEntity) UpdateTemplate(clientId string, id primitive.ObjectID, template entities.MessageTemplate) error {
	col, err := entity.collection(clientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"textTh":           template.TextTh,
		"textEn":           template.TextEn,
		"channelOverrides": template.ChannelOverrides,
		"active":           template.Active,
		"updatedBy":        template.UpdatedBy,
		"updatedAt":        time.Now(),
	}})
	return err
}

func (entity *messageTemplateEntity) SetActive(clientId string, id primitive.ObjectID, active bool, updatedBy string) error {
	col, err := entity.collection(clientId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"active":    active,
		"updatedBy": updatedBy,
		"updatedAt": time.Now(),
	}})
	return err
}

func (entity *messageTemplateEntity) CountActiveByCode(clientId string, code string, excludeId *primitive.ObjectID) (int64, error) {
	col, err := entity.collection(clientId)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	filter := bson.M{"clientId": clientId, "code": code, "active": true}
	if excludeId != nil {
		filter["_id"] = bson.M{"$ne": *excludeId}
	}
	return col.CountDocuments(ctx, filter)
}
