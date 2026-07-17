package repositories

import (
	"context"
	"fmt"
	"time"

	"alert/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type counterEntity struct {
	mongo *db.Manager
}

type ICounter interface {
	NextSequence(clientId string, prefix string, date time.Time) (int64, error)
}

func NewCounterEntity(resource *db.Resource) ICounter {
	return &counterEntity{mongo: resource.Mongo}
}

func (entity *counterEntity) NextSequence(clientId string, prefix string, date time.Time) (int64, error) {
	col, err := entity.mongo.CollectionFor(clientId, db.CollectionCounters)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	key := fmt.Sprintf("%s:%s", prefix, date.Format("060102"))
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var counter struct {
		Seq int64 `bson:"seq"`
	}
	err = col.FindOneAndUpdate(ctx,
		bson.M{"_id": key},
		bson.M{"$inc": bson.M{"seq": 1}},
		opts).Decode(&counter)
	return counter.Seq, err
}
