package messaging

import "os"

type ProviderConfig struct {
	SmsApiUrl         string
	SmsBalanceUrl     string
	SmsApiKey         string
	SmsApiSecret      string
	SmsSenderId       string
	SmsWebhookSecret  string
	LineChannelToken  string
	LineChannelSecret string
}

func EnvProviderConfig() ProviderConfig {
	return ProviderConfig{
		SmsApiUrl:         os.Getenv("SMS_API_URL"),
		SmsBalanceUrl:     os.Getenv("SMS_BALANCE_URL"),
		SmsApiKey:         os.Getenv("SMS_API_KEY"),
		SmsApiSecret:      os.Getenv("SMS_API_SECRET"),
		SmsSenderId:       os.Getenv("SMS_SENDER_ID"),
		SmsWebhookSecret:  os.Getenv("SMS_WEBHOOK_SECRET"),
		LineChannelToken:  os.Getenv("LINE_CHANNEL_TOKEN"),
		LineChannelSecret: os.Getenv("LINE_CHANNEL_SECRET"),
	}
}

func (c ProviderConfig) MergedOver(fallback ProviderConfig) ProviderConfig {
	return ProviderConfig{
		SmsApiUrl:         firstNonEmpty(c.SmsApiUrl, fallback.SmsApiUrl),
		SmsBalanceUrl:     firstNonEmpty(c.SmsBalanceUrl, fallback.SmsBalanceUrl),
		SmsApiKey:         firstNonEmpty(c.SmsApiKey, fallback.SmsApiKey),
		SmsApiSecret:      firstNonEmpty(c.SmsApiSecret, fallback.SmsApiSecret),
		SmsSenderId:       firstNonEmpty(c.SmsSenderId, fallback.SmsSenderId),
		SmsWebhookSecret:  firstNonEmpty(c.SmsWebhookSecret, fallback.SmsWebhookSecret),
		LineChannelToken:  firstNonEmpty(c.LineChannelToken, fallback.LineChannelToken),
		LineChannelSecret: firstNonEmpty(c.LineChannelSecret, fallback.LineChannelSecret),
	}
}

func (c ProviderConfig) HasSms() bool {
	return c.SmsApiUrl != "" && c.SmsApiKey != "" && c.SmsSenderId != ""
}

func (c ProviderConfig) HasLine() bool {
	return c.LineChannelToken != ""
}

func firstNonEmpty(primary string, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}
