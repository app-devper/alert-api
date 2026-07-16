package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DeliveryLog struct {
	Id                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	EventId           primitive.ObjectID `bson:"eventId" json:"eventId"`
	CheckInId         primitive.ObjectID `bson:"checkInId" json:"checkInId"`
	ClientId          string             `bson:"clientId" json:"clientId"`
	BranchId          string             `bson:"branchId" json:"branchId"`
	Channel           string             `bson:"channel" json:"channel"`
	Target            string             `bson:"target" json:"target"`
	Status            string             `bson:"status" json:"status"`
	ProviderStatus    string             `bson:"providerStatus" json:"providerStatus"`
	ProviderReference string             `bson:"providerReference" json:"providerReference"`
	QueuedAt          time.Time          `bson:"queuedAt" json:"queuedAt"`
	SentAt            *time.Time         `bson:"sentAt" json:"sentAt"`
	DeliveredAt       *time.Time         `bson:"deliveredAt" json:"deliveredAt"`
	FailReason        string             `bson:"failReason" json:"failReason"`
	ExpiresAt         time.Time          `bson:"expiresAt" json:"expiresAt"`
}
