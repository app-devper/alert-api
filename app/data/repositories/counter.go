package repositories

import (
	"context"
	"fmt"
	"time"

	"alert/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type counterEntity struct {
	col *mongo.Collection
}

type ICounter interface {
	NextSequence(clientId string, prefix string, date time.Time) (int64, error)
}

func NewCounterEntity(resource *db.Resource) ICounter {
	return &counterEntity{col: resource.AlertDb.Collection("counters")}
}

func (entity *counterEntity) NextSequence(clientId string, prefix string, date time.Time) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	key := fmt.Sprintf("%s:%s:%s", clientId, prefix, date.Format("060102"))
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var counter struct {
		Seq int64 `bson:"seq"`
	}
	err := entity.col.FindOneAndUpdate(ctx,
		bson.M{"_id": key},
		bson.M{"$inc": bson.M{"seq": 1}},
		opts).Decode(&counter)
	return counter.Seq, err
}
