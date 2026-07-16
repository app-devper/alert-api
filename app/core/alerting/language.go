package alerting

import (
	"strings"

	"alert/app/core/constant"
	"alert/app/data/entities"
)

func NormalizeLanguage(language string) string {
	if strings.EqualFold(language, constant.LanguageEn) {
		return constant.LanguageEn
	}
	return constant.LanguageTh
}

func MessageFor(template entities.MessageTemplate, preferredLanguage string, channel string) string {
	language := NormalizeLanguage(preferredLanguage)
	if override, ok := template.ChannelOverrides[strings.ToUpper(channel)]; ok {
		if text := textByLanguage(override.TextTh, override.TextEn, language); text != "" {
			return text
		}
	}
	return textByLanguage(template.TextTh, template.TextEn, language)
}

func textByLanguage(textTh string, textEn string, language string) string {
	if language == constant.LanguageEn && textEn != "" {
		return textEn
	}
	return textTh
}
