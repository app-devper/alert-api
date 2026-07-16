package alerting

import (
	"time"

	"alert/app/core/constant"
)

func NormalizeCooldownSeconds(seconds int) int {
	if seconds < constant.MinCooldownSeconds {
		return constant.DefaultCooldownSeconds
	}
	if seconds > constant.MaxCooldownSeconds {
		return constant.MaxCooldownSeconds
	}
	return seconds
}

func CooldownRemaining(lastSentAt *time.Time, now time.Time, cooldownSeconds int) time.Duration {
	if lastSentAt == nil {
		return 0
	}
	elapsed := now.Sub(*lastSentAt)
	window := time.Duration(NormalizeCooldownSeconds(cooldownSeconds)) * time.Second
	if elapsed >= window {
		return 0
	}
	return window - elapsed
}

func IsCooldownExempt(eventType string) bool {
	return eventType == constant.EventAllClear || eventType == constant.EventTest
}
