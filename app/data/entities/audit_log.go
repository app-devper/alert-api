package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuditLog struct {
	Id         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ClientId   string             `bson:"clientId" json:"clientId"`
	BranchId   string             `bson:"branchId" json:"branchId"`
	Actor      string             `bson:"actor" json:"actor"`
	Action     string             `bson:"action" json:"action"`
	Detail     bson.M             `bson:"detail" json:"detail"`
	Result     string             `bson:"result" json:"result"`
	OccurredAt time.Time          `bson:"occurredAt" json:"occurredAt"`
}
