package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type QrToken struct {
	Id        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ClientId  string             `bson:"clientId" json:"clientId"`
	BranchId  string             `bson:"branchId" json:"branchId"`
	TableNo   string             `bson:"tableNo" json:"tableNo"`
	Token     string             `bson:"token" json:"token"`
	Active    bool               `bson:"active" json:"active"`
	CreatedBy string             `bson:"createdBy" json:"createdBy"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	RevokedAt *time.Time         `bson:"revokedAt" json:"revokedAt"`
}
