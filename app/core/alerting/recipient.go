package alerting

import (
	"time"

	"alert/app/data/entities"
)

func IsEligibleRecipient(checkIn entities.CheckIn, now time.Time) bool {
	if checkIn.OtpVerifiedAt == nil {
		return false
	}
	if checkIn.CheckedOutAt != nil {
		return false
	}
	return checkIn.ExpiresAt.After(now)
}

func FilterEligibleRecipients(checkIns []entities.CheckIn, now time.Time) []entities.CheckIn {
	eligible := make([]entities.CheckIn, 0, len(checkIns))
	for _, checkIn := range checkIns {
		if IsEligibleRecipient(checkIn, now) {
			eligible = append(eligible, checkIn)
		}
	}
	return eligible
}
