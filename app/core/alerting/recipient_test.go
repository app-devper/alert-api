package alerting

import (
	"testing"
	"time"

	"alert/app/data/entities"
)

func activeCheckIn(now time.Time) entities.CheckIn {
	verified := now.Add(-time.Hour)
	return entities.CheckIn{
		OtpVerifiedAt: &verified,
		CheckedOutAt:  nil,
		ExpiresAt:     now.Add(time.Hour),
	}
}

func TestEligibleRecipientRequiresOtpVerification(t *testing.T) {
	now := time.Now()
	checkIn := activeCheckIn(now)
	checkIn.OtpVerifiedAt = nil

	if IsEligibleRecipient(checkIn, now) {
		t.Fatal("unverified check-in must not receive alerts")
	}
}

func TestEligibleRecipientExcludesCheckedOut(t *testing.T) {
	now := time.Now()
	checkIn := activeCheckIn(now)
	checkedOut := now.Add(-time.Minute)
	checkIn.CheckedOutAt = &checkedOut

	if IsEligibleRecipient(checkIn, now) {
		t.Fatal("checked-out customer must not receive alerts")
	}
}

func TestEligibleRecipientExcludesExpired(t *testing.T) {
	now := time.Now()
	checkIn := activeCheckIn(now)
	checkIn.ExpiresAt = now.Add(-time.Second)

	if IsEligibleRecipient(checkIn, now) {
		t.Fatal("expired check-in must not receive alerts")
	}
}

func TestEligibleRecipientAcceptsActive(t *testing.T) {
	now := time.Now()

	if !IsEligibleRecipient(activeCheckIn(now), now) {
		t.Fatal("active verified check-in must receive alerts")
	}
}

func TestFilterEligibleRecipientsKeepsOnlyActive(t *testing.T) {
	now := time.Now()
	expired := activeCheckIn(now)
	expired.ExpiresAt = now.Add(-time.Minute)
	unverified := activeCheckIn(now)
	unverified.OtpVerifiedAt = nil

	filtered := FilterEligibleRecipients([]entities.CheckIn{activeCheckIn(now), expired, unverified}, now)

	if len(filtered) != 1 {
		t.Fatalf("expected 1 eligible recipient, got %d", len(filtered))
	}
}

func TestFilterEligibleRecipientsEmptyInput(t *testing.T) {
	filtered := FilterEligibleRecipients(nil, time.Now())

	if len(filtered) != 0 {
		t.Fatalf("expected empty result, got %d", len(filtered))
	}
}
