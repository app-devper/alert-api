package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChannelStat struct {
	Sent      int `bson:"sent" json:"sent"`
	Delivered int `bson:"delivered" json:"delivered"`
	Failed    int `bson:"failed" json:"failed"`
}

type ChannelSummary struct {
	Sms  ChannelStat `bson:"sms" json:"sms"`
	Push ChannelStat `bson:"push" json:"push"`
	Line ChannelStat `bson:"line" json:"line"`
}

type EmergencyEvent struct {
	Id                 primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	EventNo            string             `bson:"eventNo" json:"eventNo"`
	ClientId           string             `bson:"clientId" json:"clientId"`
	BranchId           string             `bson:"branchId" json:"branchId"`
	EventType          string             `bson:"eventType" json:"eventType"`
	TemplateId         primitive.ObjectID `bson:"templateId" json:"templateId"`
	TriggeredBy        string             `bson:"triggeredBy" json:"triggeredBy"`
	ConfirmedWith      string             `bson:"confirmedWith" json:"confirmedWith"`
	CooldownOverridden bool               `bson:"cooldownOverridden" json:"cooldownOverridden"`
	RecipientCount     int                `bson:"recipientCount" json:"recipientCount"`
	ChannelSummary     ChannelSummary     `bson:"channelSummary" json:"channelSummary"`
	ProviderReference  string             `bson:"providerReference" json:"providerReference"`
	Status             string             `bson:"status" json:"status"`
	SentAt             time.Time          `bson:"sentAt" json:"sentAt"`
	ClosedAt           *time.Time         `bson:"closedAt" json:"closedAt"`
}
