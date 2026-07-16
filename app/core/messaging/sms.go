package messaging

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"alert/app/core/constant"

	"github.com/sirupsen/logrus"
)

const smsBatchSize = 100
const smsMaxRetries = 3

type smsProvider struct {
	apiUrl     string
	balanceUrl string
	apiKey     string
	apiSecret  string
	senderId   string
	client     *http.Client
}

func NewSmsProvider() MessageProvider {
	return &smsProvider{
		apiUrl:     os.Getenv("SMS_API_URL"),
		balanceUrl: os.Getenv("SMS_BALANCE_URL"),
		apiKey:     os.Getenv("SMS_API_KEY"),
		apiSecret:  os.Getenv("SMS_API_SECRET"),
		senderId:   os.Getenv("SMS_SENDER_ID"),
		client:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *smsProvider) Channel() string {
	return constant.ChannelSms
}

func (p *smsProvider) isConfigured() bool {
	return p.apiUrl != "" && p.apiKey != "" && p.senderId != ""
}

func (p *smsProvider) Send(messages []OutboundMessage) []SendResult {
	if !p.isConfigured() {
		logrus.Warn("sms provider not configured, logging only")
		return simulateSuccess(messages, "SMS")
	}
	results := make([]SendResult, 0, len(messages))
	for _, batch := range groupByText(messages, smsBatchSize) {
		results = append(results, p.sendBatch(batch)...)
	}
	return results
}

type smsApiResponse struct {
	BatchReference string `json:"batch_reference"`
	Messages       []struct {
		To        string `json:"to"`
		MessageId string `json:"message_id"`
		Status    string `json:"status"`
	} `json:"messages"`
}

func (p *smsProvider) sendBatch(batch []OutboundMessage) []SendResult {
	targets := make([]string, 0, len(batch))
	for _, message := range batch {
		targets = append(targets, message.Target)
	}
	payload, err := json.Marshal(map[string]interface{}{
		"sender":  p.senderId,
		"msisdn":  targets,
		"message": batch[0].Text,
	})
	if err != nil {
		return failAll(batch, err.Error())
	}

	body, err := p.postWithRetry(p.apiUrl, payload)
	if err != nil {
		return failAll(batch, err.Error())
	}

	var parsed smsApiResponse
	referenceByTarget := map[string]string{}
	batchReference := fmt.Sprintf("batch-%d", time.Now().UnixNano())
	if jsonErr := json.Unmarshal(body, &parsed); jsonErr == nil {
		if parsed.BatchReference != "" {
			batchReference = parsed.BatchReference
		}
		for _, m := range parsed.Messages {
			referenceByTarget[m.To] = m.MessageId
		}
	}

	results := make([]SendResult, 0, len(batch))
	for i, message := range batch {
		reference := referenceByTarget[message.Target]
		if reference == "" {
			reference = fmt.Sprintf("%s-%d", batchReference, i)
		}
		results = append(results, SendResult{
			RecipientKey:      message.RecipientKey,
			Success:           true,
			ProviderReference: reference,
			ProviderStatus:    "ACCEPTED",
		})
	}
	return results
}

func (p *smsProvider) postWithRetry(url string, payload []byte) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= smsMaxRetries; attempt++ {
		body, status, err := p.post(url, payload)
		if err == nil && status < 500 {
			if status >= 400 {
				return nil, fmt.Errorf("sms gateway rejected request: %d %s", status, string(body))
			}
			return body, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("sms gateway error: %d", status)
		}
		if attempt < smsMaxRetries {
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
	}
	return nil, lastErr
}

func (p *smsProvider) post(url string, payload []byte) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(p.apiKey, p.apiSecret)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	return buf.Bytes(), resp.StatusCode, nil
}

func (p *smsProvider) Balance() (int64, error) {
	if !p.isConfigured() || p.balanceUrl == "" {
		return 0, errors.New("sms balance endpoint not configured")
	}
	req, err := http.NewRequest(http.MethodGet, p.balanceUrl, nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(p.apiKey, p.apiSecret)
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var parsed struct {
		Credit int64 `json:"credit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, err
	}
	return parsed.Credit, nil
}

func groupByText(messages []OutboundMessage, batchSize int) [][]OutboundMessage {
	byText := map[string][]OutboundMessage{}
	order := []string{}
	for _, message := range messages {
		if _, seen := byText[message.Text]; !seen {
			order = append(order, message.Text)
		}
		byText[message.Text] = append(byText[message.Text], message)
	}
	batches := [][]OutboundMessage{}
	for _, text := range order {
		group := byText[text]
		for start := 0; start < len(group); start += batchSize {
			end := start + batchSize
			if end > len(group) {
				end = len(group)
			}
			batches = append(batches, group[start:end])
		}
	}
	return batches
}

func simulateSuccess(messages []OutboundMessage, channel string) []SendResult {
	results := make([]SendResult, 0, len(messages))
	for i, message := range messages {
		logrus.Infof("[dev-%s] to=%s text=%s", channel, message.Target, message.Text)
		results = append(results, SendResult{
			RecipientKey:      message.RecipientKey,
			Success:           true,
			ProviderReference: fmt.Sprintf("dev-%s-%d-%d", channel, time.Now().UnixNano(), i),
			ProviderStatus:    "SIMULATED",
		})
	}
	return results
}
