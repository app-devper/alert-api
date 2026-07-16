package messaging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"alert/app/core/constant"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/sirupsen/logrus"
)

const pushConcurrency = 20

type pushProvider struct {
	vapidPublicKey  string
	vapidPrivateKey string
	subscriber      string
}

type pushMessagePayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func NewPushProvider() MessageProvider {
	return &pushProvider{
		vapidPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		vapidPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		subscriber:      os.Getenv("VAPID_SUBSCRIBER"),
	}
}

func (p *pushProvider) Channel() string {
	return constant.ChannelPush
}

func (p *pushProvider) isConfigured() bool {
	return p.vapidPublicKey != "" && p.vapidPrivateKey != ""
}

func (p *pushProvider) Send(messages []OutboundMessage) []SendResult {
	if !p.isConfigured() {
		logrus.Warn("push provider not configured, logging only")
		return simulateSuccess(messages, "PUSH")
	}
	results := make([]SendResult, len(messages))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, pushConcurrency)
	for i, message := range messages {
		wg.Add(1)
		go func(index int, msg OutboundMessage) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			results[index] = p.sendOne(msg)
		}(i, message)
	}
	wg.Wait()
	return results
}

func (p *pushProvider) sendOne(message OutboundMessage) SendResult {
	if message.Push == nil || message.Push.Endpoint == "" {
		return SendResult{RecipientKey: message.RecipientKey, FailReason: "missing push subscription"}
	}

	payload, err := json.Marshal(pushMessagePayload{Title: "แจ้งเตือนฉุกเฉิน / Emergency Alert", Body: message.Text})
	if err != nil {
		return SendResult{RecipientKey: message.RecipientKey, FailReason: err.Error()}
	}

	subscription := &webpush.Subscription{
		Endpoint: message.Push.Endpoint,
		Keys:     webpush.Keys{P256dh: message.Push.P256dh, Auth: message.Push.Auth},
	}
	resp, err := webpush.SendNotification(payload, subscription, &webpush.Options{
		Subscriber:      p.subscriber,
		VAPIDPublicKey:  p.vapidPublicKey,
		VAPIDPrivateKey: p.vapidPrivateKey,
		TTL:             300,
		Urgency:         webpush.UrgencyHigh,
	})
	if err != nil {
		return SendResult{RecipientKey: message.RecipientKey, FailReason: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		return SendResult{
			RecipientKey:     message.RecipientKey,
			FailReason:       fmt.Sprintf("subscription gone: %d", resp.StatusCode),
			SubscriptionGone: true,
		}
	}
	if resp.StatusCode >= 400 {
		return SendResult{RecipientKey: message.RecipientKey, FailReason: fmt.Sprintf("push error: %d", resp.StatusCode)}
	}
	return SendResult{
		RecipientKey:      message.RecipientKey,
		Success:           true,
		ProviderReference: fmt.Sprintf("push-%d", time.Now().UnixNano()),
		ProviderStatus:    fmt.Sprintf("%d", resp.StatusCode),
	}
}
