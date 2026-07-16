package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChannelText struct {
	TextTh string `bson:"textTh" json:"textTh"`
	TextEn string `bson:"textEn" json:"textEn"`
}

type MessageTemplate struct {
	Id               primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	ClientId         string                 `bson:"clientId" json:"clientId"`
	Code             string                 `bson:"code" json:"code"`
	TextTh           string                 `bson:"textTh" json:"textTh"`
	TextEn           string                 `bson:"textEn" json:"textEn"`
	ChannelOverrides map[string]ChannelText `bson:"channelOverrides" json:"channelOverrides"`
	Active           bool                   `bson:"active" json:"active"`
	UpdatedBy        string                 `bson:"updatedBy" json:"updatedBy"`
	UpdatedAt        time.Time              `bson:"updatedAt" json:"updatedAt"`
}
