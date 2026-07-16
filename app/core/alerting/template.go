package alerting

import (
	"errors"
	"regexp"
	"unicode"
)

const GsmSingleSegmentLength = 160
const UcsSingleSegmentLength = 70
const GsmMultiSegmentLength = 153
const UcsMultiSegmentLength = 67

var linkPattern = regexp.MustCompile(`(?i)(https?://|www\.|[a-z0-9-]+\.(com|net|org|io|co|th|app|me|ly|link|info|biz|dev)(/|\b))`)

func ValidateNoLink(text string) error {
	if linkPattern.MatchString(text) {
		return errors.New("emergency message must not contain links")
	}
	return nil
}

func SmsSegmentCount(text string) int {
	runes := []rune(text)
	if len(runes) == 0 {
		return 0
	}
	single, multi := GsmSingleSegmentLength, GsmMultiSegmentLength
	if requiresUcs2(runes) {
		single, multi = UcsSingleSegmentLength, UcsMultiSegmentLength
	}
	if len(runes) <= single {
		return 1
	}
	return (len(runes) + multi - 1) / multi
}

func requiresUcs2(runes []rune) bool {
	for _, r := range runes {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}
