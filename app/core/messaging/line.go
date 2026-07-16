package messaging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"alert/app/core/constant"

	"github.com/sirupsen/logrus"
)

const linePnpPushUrl = "https://api.line.me/bot/pnp/push"
const lineConcurrency = 20

type lineProvider struct {
	channelToken string
	client       *http.Client
}

func NewLineProvider() MessageProvider {
	return &lineProvider{
		channelToken: os.Getenv("LINE_CHANNEL_TOKEN"),
		client:       &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *lineProvider) Channel() string {
	return constant.ChannelLine
}

func (p *lineProvider) Send(messages []OutboundMessage) []SendResult {
	if p.channelToken == "" {
		logrus.Warn("line provider not configured, logging only")
		return simulateSuccess(messages, "LINE")
	}
	results := make([]SendResult, len(messages))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, lineConcurrency)
	for i, message := range messages {
		wg.Add(1)
		go func(index int, msg OutboundMessage) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			results[index] = p.sendOne(msg, index)
		}(i, message)
	}
	wg.Wait()
	return results
}

func (p *lineProvider) sendOne(message OutboundMessage, index int) SendResult {
	deliveryTag := fmt.Sprintf("lon-%d-%d", time.Now().UnixNano(), index)
	payload, err := json.Marshal(map[string]interface{}{
		"to": message.Target,
		"messages": []map[string]string{
			{"type": "text", "text": message.Text},
		},
	})
	if err != nil {
		return SendResult{RecipientKey: message.RecipientKey, FailReason: err.Error()}
	}
	req, err := http.NewRequest(http.MethodPost, linePnpPushUrl, bytes.NewReader(payload))
	if err != nil {
		return SendResult{RecipientKey: message.RecipientKey, FailReason: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.channelToken)
	req.Header.Set("X-Line-Delivery-Tag", deliveryTag)
	resp, err := p.client.Do(req)
	if err != nil {
		return SendResult{RecipientKey: message.RecipientKey, FailReason: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnprocessableEntity {
		return SendResult{
			RecipientKey:   message.RecipientKey,
			FailReason:     "phone number not reachable on LINE",
			ProviderStatus: fmt.Sprintf("%d", resp.StatusCode),
		}
	}
	if resp.StatusCode >= 400 {
		return SendResult{
			RecipientKey:   message.RecipientKey,
			FailReason:     fmt.Sprintf("line pnp error: %d", resp.StatusCode),
			ProviderStatus: fmt.Sprintf("%d", resp.StatusCode),
		}
	}
	return SendResult{
		RecipientKey:      message.RecipientKey,
		Success:           true,
		ProviderReference: deliveryTag,
		ProviderStatus:    fmt.Sprintf("%d", resp.StatusCode),
	}
}
