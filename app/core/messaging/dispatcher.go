package messaging

import (
	"sync"
	"time"

	"alert/app/core/alerting"
	"alert/app/core/constant"
	"alert/app/data/entities"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Dispatcher struct {
	providers map[string]MessageProvider
}

type DispatchOutcome struct {
	Logs                []entities.DeliveryLog
	Summary             entities.ChannelSummary
	GoneSubscriptionIds []primitive.ObjectID
	ProviderReference   string
}

func NewDispatcher(providers ...MessageProvider) *Dispatcher {
	providerMap := make(map[string]MessageProvider, len(providers))
	for _, provider := range providers {
		providerMap[provider.Channel()] = provider
	}
	return &Dispatcher{providers: providerMap}
}

func (d *Dispatcher) DispatchAlert(cfg ProviderConfig, event entities.EmergencyEvent, recipients []entities.CheckIn, template entities.MessageTemplate, logTtl time.Duration) DispatchOutcome {
	messagesByChannel := buildMessages(cfg, event.ClientId, recipients, template)
	return d.dispatch(cfg, event, messagesByChannel, logTtl)
}

func (d *Dispatcher) DispatchTest(cfg ProviderConfig, event entities.EmergencyEvent, testRecipients []entities.StaffPermission, template entities.MessageTemplate, logTtl time.Duration) DispatchOutcome {
	if !cfg.SmsEnabled {
		return DispatchOutcome{}
	}
	messages := make([]OutboundMessage, 0, len(testRecipients))
	for _, recipient := range testRecipients {
		messages = append(messages, OutboundMessage{
			RecipientKey: recipient.Id.Hex(),
			TenantId:     event.ClientId,
			Target:       recipient.Phone,
			Text:         alerting.MessageFor(template, constant.LanguageTh, constant.ChannelSms),
		})
	}
	return d.dispatch(cfg, event, map[string][]OutboundMessage{constant.ChannelSms: messages}, logTtl)
}

func buildMessages(cfg ProviderConfig, clientId string, recipients []entities.CheckIn, template entities.MessageTemplate) map[string][]OutboundMessage {
	byChannel := map[string][]OutboundMessage{}
	for _, recipient := range recipients {
		key := recipient.Id.Hex()
		if cfg.SmsEnabled {
			byChannel[constant.ChannelSms] = append(byChannel[constant.ChannelSms], OutboundMessage{
				RecipientKey: key,
				TenantId:     clientId,
				Target:       recipient.Phone,
				Text:         alerting.MessageFor(template, recipient.PreferredLanguage, constant.ChannelSms),
			})
		}
		if recipient.HasPush() {
			byChannel[constant.ChannelPush] = append(byChannel[constant.ChannelPush], OutboundMessage{
				RecipientKey: key,
				TenantId:     clientId,
				Target:       recipient.PushSubscription.Endpoint,
				Text:         alerting.MessageFor(template, recipient.PreferredLanguage, constant.ChannelPush),
				Push: &PushTarget{
					Endpoint: recipient.PushSubscription.Endpoint,
					P256dh:   recipient.PushSubscription.Keys.P256dh,
					Auth:     recipient.PushSubscription.Keys.Auth,
				},
			})
		}
		if cfg.LineEnabled {
			byChannel[constant.ChannelLine] = append(byChannel[constant.ChannelLine], OutboundMessage{
				RecipientKey: key,
				TenantId:     clientId,
				Target:       recipient.Phone,
				Text:         alerting.MessageFor(template, recipient.PreferredLanguage, constant.ChannelLine),
			})
		}
	}
	return byChannel
}

func (d *Dispatcher) dispatch(cfg ProviderConfig, event entities.EmergencyEvent, messagesByChannel map[string][]OutboundMessage, logTtl time.Duration) DispatchOutcome {
	type channelResult struct {
		channel  string
		messages []OutboundMessage
		results  []SendResult
	}

	var wg sync.WaitGroup
	resultCh := make(chan channelResult, len(messagesByChannel))
	for channel, messages := range messagesByChannel {
		provider, ok := d.providers[channel]
		if !ok || len(messages) == 0 {
			continue
		}
		wg.Add(1)
		go func(ch string, p MessageProvider, msgs []OutboundMessage) {
			defer wg.Done()
			resultCh <- channelResult{channel: ch, messages: msgs, results: p.Send(cfg, msgs)}
		}(channel, provider, messages)
	}
	wg.Wait()
	close(resultCh)

	now := time.Now()
	outcome := DispatchOutcome{}
	for result := range resultCh {
		targetsByKey := map[string]string{}
		for _, message := range result.messages {
			targetsByKey[message.RecipientKey] = message.Target
		}
		for _, sendResult := range result.results {
			logEntry := buildDeliveryLog(event, result.channel, sendResult, targetsByKey[sendResult.RecipientKey], now, logTtl)
			outcome.Logs = append(outcome.Logs, logEntry)
			applyToSummary(&outcome.Summary, result.channel, sendResult.Success)
			if sendResult.Success && result.channel == constant.ChannelSms && outcome.ProviderReference == "" {
				outcome.ProviderReference = sendResult.ProviderReference
			}
			if sendResult.SubscriptionGone {
				if id, err := primitive.ObjectIDFromHex(sendResult.RecipientKey); err == nil {
					outcome.GoneSubscriptionIds = append(outcome.GoneSubscriptionIds, id)
				}
			}
		}
	}
	return outcome
}

func buildDeliveryLog(event entities.EmergencyEvent, channel string, sendResult SendResult, target string, now time.Time, logTtl time.Duration) entities.DeliveryLog {
	logEntry := entities.DeliveryLog{
		Id:                primitive.NewObjectID(),
		EventId:           event.Id,
		ClientId:          event.ClientId,
		BranchId:          event.BranchId,
		Channel:           channel,
		Target:            maskTarget(channel, target),
		ProviderStatus:    sendResult.ProviderStatus,
		ProviderReference: sendResult.ProviderReference,
		QueuedAt:          now,
		ExpiresAt:         now.Add(logTtl),
	}
	if id, err := primitive.ObjectIDFromHex(sendResult.RecipientKey); err == nil {
		logEntry.CheckInId = id
	}
	if sendResult.Success {
		logEntry.Status = constant.DeliverySent
		sentAt := now
		logEntry.SentAt = &sentAt
	} else {
		logEntry.Status = constant.DeliveryFailed
		logEntry.FailReason = sendResult.FailReason
	}
	return logEntry
}

func maskTarget(channel string, target string) string {
	if channel == constant.ChannelPush {
		return "push:" + alerting.HashToken(target)[:16]
	}
	return alerting.MaskPhone(target)
}

func applyToSummary(summary *entities.ChannelSummary, channel string, success bool) {
	var stat *entities.ChannelStat
	switch channel {
	case constant.ChannelSms:
		stat = &summary.Sms
	case constant.ChannelPush:
		stat = &summary.Push
	case constant.ChannelLine:
		stat = &summary.Line
	default:
		return
	}
	if success {
		stat.Sent++
	} else {
		stat.Failed++
	}
}
