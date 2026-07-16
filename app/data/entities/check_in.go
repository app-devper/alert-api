package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PushKeys struct {
	P256dh string `bson:"p256dh" json:"p256dh"`
	Auth   string `bson:"auth" json:"auth"`
}

type PushSubscription struct {
	Endpoint string   `bson:"endpoint" json:"endpoint"`
	Keys     PushKeys `bson:"keys" json:"keys"`
}

type CheckIn struct {
	Id                   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CheckInNo            string             `bson:"checkInNo" json:"checkInNo"`
	ClientId             string             `bson:"clientId" json:"clientId"`
	BranchId             string             `bson:"branchId" json:"branchId"`
	Phone                string             `bson:"phone" json:"-"`
	GroupSize            int                `bson:"groupSize" json:"groupSize"`
	TableNo              string             `bson:"tableNo" json:"tableNo"`
	PreferredLanguage    string             `bson:"preferredLanguage" json:"preferredLanguage"`
	MarketingConsent     bool               `bson:"marketingConsent" json:"marketingConsent"`
	ConsentAt            time.Time          `bson:"consentAt" json:"consentAt"`
	PrivacyNoticeVersion string             `bson:"privacyNoticeVersion" json:"privacyNoticeVersion"`
	OtpVerifiedAt        *time.Time         `bson:"otpVerifiedAt" json:"otpVerifiedAt"`
	PushSubscription     *PushSubscription  `bson:"pushSubscription" json:"-"`
	LineUserId           string             `bson:"lineUserId" json:"-"`
	SessionTokenHash     string             `bson:"sessionTokenHash" json:"-"`
	CheckedInAt          time.Time          `bson:"checkedInAt" json:"checkedInAt"`
	CheckedOutAt         *time.Time         `bson:"checkedOutAt" json:"checkedOutAt"`
	CheckedOutBy         string             `bson:"checkedOutBy" json:"checkedOutBy"`
	ExpiresAt            time.Time          `bson:"expiresAt" json:"expiresAt"`
}

func (c CheckIn) HasPush() bool {
	return c.PushSubscription != nil && c.PushSubscription.Endpoint != ""
}

func (c CheckIn) HasLine() bool {
	return c.LineUserId != ""
}
