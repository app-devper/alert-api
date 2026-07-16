package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OtpRequest struct {
	Id           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ClientId     string             `bson:"clientId" json:"clientId"`
	CheckInId    primitive.ObjectID `bson:"checkInId" json:"checkInId"`
	Phone        string             `bson:"phone" json:"-"`
	OtpHash      string             `bson:"otpHash" json:"-"`
	RefCode      string             `bson:"refCode" json:"refCode"`
	AttemptCount int                `bson:"attemptCount" json:"attemptCount"`
	VerifiedAt   *time.Time         `bson:"verifiedAt" json:"verifiedAt"`
	ExpiresAt    time.Time          `bson:"expiresAt" json:"expiresAt"`
}
