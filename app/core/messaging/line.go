package messaging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"alert/app/core/constant"

	"github.com/sirupsen/logrus"
)

const lineMulticastUrl = "https://api.line.me/v2/bot/message/multicast"
const lineMulticastBatchSize = 500

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
	results := make([]SendResult, 0, len(messages))
	for _, batch := range groupByText(messages, lineMulticastBatchSize) {
		results = append(results, p.sendBatch(batch)...)
	}
	return results
}

func (p *lineProvider) sendBatch(batch []OutboundMessage) []SendResult {
	userIds := make([]string, 0, len(batch))
	for _, message := range batch {
		userIds = append(userIds, message.Target)
	}
	payload, err := json.Marshal(map[string]interface{}{
		"to": userIds,
		"messages": []map[string]string{
			{"type": "text", "text": batch[0].Text},
		},
	})
	if err != nil {
		return failAll(batch, err.Error())
	}
	req, err := http.NewRequest(http.MethodPost, lineMulticastUrl, bytes.NewReader(payload))
	if err != nil {
		return failAll(batch, err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.channelToken)
	resp, err := p.client.Do(req)
	if err != nil {
		return failAll(batch, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return failAll(batch, fmt.Sprintf("line api error: %d", resp.StatusCode))
	}
	reference := fmt.Sprintf("line-%d", time.Now().UnixNano())
	results := make([]SendResult, 0, len(batch))
	for i, message := range batch {
		results = append(results, SendResult{
			RecipientKey:      message.RecipientKey,
			Success:           true,
			ProviderReference: fmt.Sprintf("%s-%d", reference, i),
			ProviderStatus:    fmt.Sprintf("%d", resp.StatusCode),
		})
	}
	return results
}
