package db

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultDbPrefix = "alert"

type Resource struct {
	Mongo       *Manager
	RdDb        *redis.Client
	mongoClient *mongo.Client
}

func (r *Resource) Close() {
	logrus.Warning("Closing all db connections")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if r.mongoClient != nil {
		if err := r.mongoClient.Disconnect(ctx); err != nil {
			logrus.Error("failed to disconnect mongo: ", err)
		}
	}
	if r.RdDb != nil {
		if err := r.RdDb.Close(); err != nil {
			logrus.Error("failed to close redis: ", err)
		}
	}
}

func InitResource(seeder Seeder) (*Resource, error) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Print(err)
	}

	host := os.Getenv("MONGO_HOST")
	dbPrefix := os.Getenv("MONGO_DB_PREFIX")
	if dbPrefix == "" {
		dbPrefix = defaultDbPrefix
	}
	mongoClient, err := mongo.NewClient(options.Client().ApplyURI(host))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = mongoClient.Connect(ctx)
	if err != nil {
		return nil, err
	}
	if err := mongoClient.Ping(ctx, nil); err != nil {
		return nil, err
	}

	redisHost := os.Getenv("REDIS_HOST")
	redisOp, err := redis.ParseURL(redisHost)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(redisOp)

	return &Resource{
		Mongo:       NewManager(mongoClient, dbPrefix, seeder),
		RdDb:        rdb,
		mongoClient: mongoClient,
	}, nil
}

func createTenantIndexes(ctx context.Context, database *mongo.Database) error {
	ttl := options.Index().SetExpireAfterSeconds(0)

	indexes := map[string][]mongo.IndexModel{
		CollectionCheckIns: {
			{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: ttl},
			{Keys: bson.D{{Key: "clientId", Value: 1}, {Key: "branchId", Value: 1}, {Key: "checkedOutAt", Value: 1}}},
			{Keys: bson.D{{Key: "sessionTokenHash", Value: 1}}},
		},
		CollectionOtpRequests: {
			{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
			{Keys: bson.D{{Key: "checkInId", Value: 1}}},
		},
		CollectionDeliveryLogs: {
			{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
			{Keys: bson.D{{Key: "eventId", Value: 1}}},
			{Keys: bson.D{{Key: "providerReference", Value: 1}}},
		},
		CollectionEmergencyEvents: {
			{Keys: bson.D{{Key: "clientId", Value: 1}, {Key: "branchId", Value: 1}, {Key: "eventType", Value: 1}, {Key: "sentAt", Value: -1}}},
		},
		CollectionAuditLogs: {
			{Keys: bson.D{{Key: "clientId", Value: 1}, {Key: "occurredAt", Value: -1}}},
		},
		CollectionQrTokens: {
			{Keys: bson.D{{Key: "token", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		CollectionStaffPermissions: {
			{Keys: bson.D{{Key: "clientId", Value: 1}, {Key: "userId", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		CollectionMessageTemplates: {
			{Keys: bson.D{{Key: "clientId", Value: 1}, {Key: "code", Value: 1}}},
		},
	}

	for collection, models := range indexes {
		if _, err := database.Collection(collection).Indexes().CreateMany(ctx, models); err != nil {
			return err
		}
	}
	return nil
}
