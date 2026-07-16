package messaging

type PushTarget struct {
	Endpoint string
	P256dh   string
	Auth     string
}

type OutboundMessage struct {
	RecipientKey string
	Target       string
	Text         string
	Push         *PushTarget
}

type SendResult struct {
	RecipientKey      string
	Success           bool
	ProviderReference string
	ProviderStatus    string
	FailReason        string
	SubscriptionGone  bool
}

type MessageProvider interface {
	Channel() string
	Send(messages []OutboundMessage) []SendResult
}

type BalanceChecker interface {
	Balance() (int64, error)
}

func failAll(messages []OutboundMessage, reason string) []SendResult {
	results := make([]SendResult, 0, len(messages))
	for _, message := range messages {
		results = append(results, SendResult{
			RecipientKey: message.RecipientKey,
			Success:      false,
			FailReason:   reason,
		})
	}
	return results
}
