package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BranchSetting struct {
	Id                 primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ClientId           string             `bson:"clientId" json:"clientId"`
	BranchId           string             `bson:"branchId" json:"branchId"`
	ShopName           string             `bson:"shopName" json:"shopName"`
	RetentionHours     int                `bson:"retentionHours" json:"retentionHours"`
	CooldownSeconds    int                `bson:"cooldownSeconds" json:"cooldownSeconds"`
	ConfirmMethod      string             `bson:"confirmMethod" json:"confirmMethod"`
	PinHash            string             `bson:"pinHash" json:"-"`
	SmsCreditThreshold int                `bson:"smsCreditThreshold" json:"smsCreditThreshold"`
	ContactChannel     string             `bson:"contactChannel" json:"contactChannel"`
	UpdatedBy          string             `bson:"updatedBy" json:"updatedBy"`
	UpdatedAt          time.Time          `bson:"updatedAt" json:"updatedAt"`
}

func (s BranchSetting) HasPin() bool {
	return s.PinHash != ""
}
