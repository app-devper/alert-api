package alerting

import (
	"testing"

	"alert/app/core/constant"
	"alert/app/data/entities"
)

func sampleTemplate() entities.MessageTemplate {
	return entities.MessageTemplate{
		TextTh: "ข้อความภาษาไทย",
		TextEn: "English message",
		ChannelOverrides: map[string]entities.ChannelText{
			constant.ChannelPush: {TextTh: "ไทยสั้น", TextEn: "Short EN"},
		},
	}
}

func TestMessageForDefaultsToThai(t *testing.T) {
	text := MessageFor(sampleTemplate(), "", constant.ChannelSms)

	if text != "ข้อความภาษาไทย" {
		t.Fatalf("expected thai text, got %s", text)
	}
}

func TestMessageForSelectsEnglish(t *testing.T) {
	text := MessageFor(sampleTemplate(), "EN", constant.ChannelSms)

	if text != "English message" {
		t.Fatalf("expected english text, got %s", text)
	}
}

func TestMessageForIsCaseInsensitive(t *testing.T) {
	text := MessageFor(sampleTemplate(), "en", constant.ChannelSms)

	if text != "English message" {
		t.Fatalf("expected english text, got %s", text)
	}
}

func TestMessageForUsesChannelOverride(t *testing.T) {
	text := MessageFor(sampleTemplate(), "TH", constant.ChannelPush)

	if text != "ไทยสั้น" {
		t.Fatalf("expected push override, got %s", text)
	}
}

func TestMessageForFallsBackWhenOverrideEmpty(t *testing.T) {
	template := sampleTemplate()
	template.ChannelOverrides[constant.ChannelLine] = entities.ChannelText{}

	text := MessageFor(template, "EN", constant.ChannelLine)

	if text != "English message" {
		t.Fatalf("expected fallback to main text, got %s", text)
	}
}

func TestMessageForUnknownLanguageFallsBackToThai(t *testing.T) {
	text := MessageFor(sampleTemplate(), "JP", constant.ChannelSms)

	if text != "ข้อความภาษาไทย" {
		t.Fatalf("expected thai fallback, got %s", text)
	}
}

func TestNormalizeLanguage(t *testing.T) {
	if NormalizeLanguage("en") != constant.LanguageEn {
		t.Fatal("expected EN")
	}
	if NormalizeLanguage("th") != constant.LanguageTh {
		t.Fatal("expected TH")
	}
	if NormalizeLanguage("") != constant.LanguageTh {
		t.Fatal("expected TH default")
	}
}
