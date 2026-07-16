package alerting

import (
	"errors"
	"regexp"
	"strings"
)

var thaiMobilePattern = regexp.MustCompile(`^0[689]\d{8}$`)
var normalizedPattern = regexp.MustCompile(`^\+66[689]\d{8}$`)

func NormalizeThaiPhone(input string) (string, error) {
	cleaned := strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(input))
	if normalizedPattern.MatchString(cleaned) {
		return cleaned, nil
	}
	if strings.HasPrefix(cleaned, "66") && normalizedPattern.MatchString("+"+cleaned) {
		return "+" + cleaned, nil
	}
	if thaiMobilePattern.MatchString(cleaned) {
		return "+66" + cleaned[1:], nil
	}
	return "", errors.New("invalid thai mobile number")
}

func MaskPhone(normalized string) string {
	if len(normalized) < 8 {
		return "XXXX"
	}
	return normalized[:len(normalized)-7] + "XXX" + normalized[len(normalized)-4:]
}

func MaskPhoneDisplay(normalized string) string {
	if !normalizedPattern.MatchString(normalized) {
		return MaskPhone(normalized)
	}
	local := "0" + normalized[3:]
	return local[:3] + "-XXX-" + local[len(local)-4:]
}

func PhoneLast4(normalized string) string {
	if len(normalized) < 4 {
		return normalized
	}
	return normalized[len(normalized)-4:]
}
