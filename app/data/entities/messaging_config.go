package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessagingConfig struct {
	Id                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ClientId          string             `bson:"clientId" json:"clientId"`
	SmsApiUrl         string             `bson:"smsApiUrl" json:"smsApiUrl"`
	SmsBalanceUrl     string             `bson:"smsBalanceUrl" json:"smsBalanceUrl"`
	SmsApiKey         string             `bson:"smsApiKey" json:"-"`
	SmsApiSecret      string             `bson:"smsApiSecret" json:"-"`
	SmsSenderId       string             `bson:"smsSenderId" json:"smsSenderId"`
	SmsWebhookSecret  string             `bson:"smsWebhookSecret" json:"-"`
	LineChannelToken  string             `bson:"lineChannelToken" json:"-"`
	LineChannelSecret string             `bson:"lineChannelSecret" json:"-"`
	UpdatedBy         string             `bson:"updatedBy" json:"updatedBy"`
	UpdatedAt         time.Time          `bson:"updatedAt" json:"updatedAt"`
}
