package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StaffPermission struct {
	Id                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ClientId          string             `bson:"clientId" json:"clientId"`
	UserId            string             `bson:"userId" json:"userId"`
	BranchId          string             `bson:"branchId" json:"branchId"`
	Phone             string             `bson:"phone" json:"phone"`
	AllowedEventTypes []string           `bson:"allowedEventTypes" json:"allowedEventTypes"`
	IsTestRecipient   bool               `bson:"isTestRecipient" json:"isTestRecipient"`
	Active            bool               `bson:"active" json:"active"`
	UpdatedBy         string             `bson:"updatedBy" json:"updatedBy"`
	UpdatedAt         time.Time          `bson:"updatedAt" json:"updatedAt"`
}

func (p StaffPermission) AllowsEventType(eventType string) bool {
	for _, allowed := range p.AllowedEventTypes {
		if allowed == eventType {
			return true
		}
	}
	return false
}
