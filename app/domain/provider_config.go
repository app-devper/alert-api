package domain

import (
	"alert/app/core/messaging"
	"alert/app/data/entities"
)

func (r *Repository) ProviderConfigFor(clientId string) messaging.ProviderConfig {
	config, err := r.MessagingConfig.GetConfig(clientId)
	if err != nil {
		return messaging.EnvProviderConfig()
	}
	return ProviderConfigFromEntity(config).MergedOver(messaging.EnvProviderConfig())
}

func ProviderConfigFromEntity(config entities.MessagingConfig) messaging.ProviderConfig {
	return messaging.ProviderConfig{
		SmsEnabled:        config.SmsEnabled,
		LineEnabled:       config.LineEnabled,
		SmsApiUrl:         config.SmsApiUrl,
		SmsBalanceUrl:     config.SmsBalanceUrl,
		SmsApiKey:         config.SmsApiKey,
		SmsApiSecret:      config.SmsApiSecret,
		SmsSenderId:       config.SmsSenderId,
		SmsWebhookSecret:  config.SmsWebhookSecret,
		LineChannelToken:  config.LineChannelToken,
		LineChannelSecret: config.LineChannelSecret,
	}
}
