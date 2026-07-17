package messaging

import (
	"testing"
	"time"

	"alert/app/core/constant"
	"alert/app/data/entities"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type fakeProvider struct {
	channel  string
	fail     bool
	gone     bool
	received []OutboundMessage
}

func (p *fakeProvider) Channel() string {
	return p.channel
}

func (p *fakeProvider) Send(messages []OutboundMessage) []SendResult {
	p.received = messages
	results := make([]SendResult, 0, len(messages))
	for _, message := range messages {
		results = append(results, SendResult{
			RecipientKey:      message.RecipientKey,
			Success:           !p.fail,
			ProviderReference: "ref-" + message.RecipientKey,
			FailReason:        failReason(p.fail),
			SubscriptionGone:  p.gone,
		})
	}
	return results
}

func failReason(fail bool) string {
	if fail {
		return "provider down"
	}
	return ""
}

func verifiedCheckIn(withPush bool) entities.CheckIn {
	verified := time.Now().Add(-time.Minute)
	checkIn := entities.CheckIn{
		Id:                primitive.NewObjectID(),
		Phone:             "+66812345678",
		PreferredLanguage: constant.LanguageTh,
		OtpVerifiedAt:     &verified,
		ExpiresAt:         time.Now().Add(time.Hour),
	}
	if withPush {
		checkIn.PushSubscription = &entities.PushSubscription{
			Endpoint: "https://push.example/sub1",
			Keys:     entities.PushKeys{P256dh: "key", Auth: "auth"},
		}
	}
	return checkIn
}

func sampleEvent() entities.EmergencyEvent {
	return entities.EmergencyEvent{
		Id:        primitive.NewObjectID(),
		ClientId:  "001",
		BranchId:  "HQ",
		EventType: constant.EventFire,
	}
}

func dispatchTemplate() entities.MessageTemplate {
	return entities.MessageTemplate{TextTh: "ไฟไหม้", TextEn: "Fire"}
}

func TestDispatchSendsAllChannelsInParallel(t *testing.T) {
	sms := &fakeProvider{channel: constant.ChannelSms}
	push := &fakeProvider{channel: constant.ChannelPush}
	line := &fakeProvider{channel: constant.ChannelLine}
	dispatcher := NewDispatcher(sms, push, line)

	outcome := dispatcher.DispatchAlert(sampleEvent(), []entities.CheckIn{verifiedCheckIn(true)}, dispatchTemplate(), time.Hour)

	if len(outcome.Logs) != 3 {
		t.Fatalf("expected 3 delivery logs, got %d", len(outcome.Logs))
	}
	if outcome.Summary.Sms.Sent != 1 || outcome.Summary.Push.Sent != 1 || outcome.Summary.Line.Sent != 1 {
		t.Fatalf("expected one sent per channel, got %+v", outcome.Summary)
	}
}

func TestDispatchLineTargetsPhoneNumberViaLon(t *testing.T) {
	line := &fakeProvider{channel: constant.ChannelLine}
	dispatcher := NewDispatcher(line)

	dispatcher.DispatchAlert(sampleEvent(), []entities.CheckIn{verifiedCheckIn(false)}, dispatchTemplate(), time.Hour)

	if len(line.received) != 1 {
		t.Fatalf("expected 1 line message, got %d", len(line.received))
	}
	if line.received[0].Target != "+66812345678" {
		t.Fatalf("expected phone target for LON, got %s", line.received[0].Target)
	}
}

func TestDispatchSkipsPushWithoutSubscription(t *testing.T) {
	sms := &fakeProvider{channel: constant.ChannelSms}
	push := &fakeProvider{channel: constant.ChannelPush}
	line := &fakeProvider{channel: constant.ChannelLine}
	dispatcher := NewDispatcher(sms, push, line)

	outcome := dispatcher.DispatchAlert(sampleEvent(), []entities.CheckIn{verifiedCheckIn(false)}, dispatchTemplate(), time.Hour)

	if len(outcome.Logs) != 2 {
		t.Fatalf("expected 2 delivery logs (sms + line), got %d", len(outcome.Logs))
	}
	if len(push.received) != 0 {
		t.Fatal("push provider must not receive messages")
	}
}

func TestDispatchChannelFailureDoesNotBlockOthers(t *testing.T) {
	sms := &fakeProvider{channel: constant.ChannelSms}
	line := &fakeProvider{channel: constant.ChannelLine, fail: true}
	dispatcher := NewDispatcher(sms, line)

	outcome := dispatcher.DispatchAlert(sampleEvent(), []entities.CheckIn{verifiedCheckIn(false)}, dispatchTemplate(), time.Hour)

	if outcome.Summary.Sms.Sent != 1 {
		t.Fatalf("sms must still send, got %+v", outcome.Summary)
	}
	if outcome.Summary.Line.Failed != 1 {
		t.Fatalf("line failure must be recorded, got %+v", outcome.Summary)
	}
}

func TestDispatchMasksPhoneTargetsInLogs(t *testing.T) {
	sms := &fakeProvider{channel: constant.ChannelSms}
	line := &fakeProvider{channel: constant.ChannelLine}
	dispatcher := NewDispatcher(sms, line)

	outcome := dispatcher.DispatchAlert(sampleEvent(), []entities.CheckIn{verifiedCheckIn(false)}, dispatchTemplate(), time.Hour)

	for _, logEntry := range outcome.Logs {
		if logEntry.Target != "+6681XXX5678" {
			t.Fatalf("expected masked target on %s, got %s", logEntry.Channel, logEntry.Target)
		}
	}
}

func TestDispatchCollectsGoneSubscriptions(t *testing.T) {
	push := &fakeProvider{channel: constant.ChannelPush, fail: true, gone: true}
	dispatcher := NewDispatcher(push)
	recipient := verifiedCheckIn(true)

	outcome := dispatcher.DispatchAlert(sampleEvent(), []entities.CheckIn{recipient}, dispatchTemplate(), time.Hour)

	if len(outcome.GoneSubscriptionIds) != 1 || outcome.GoneSubscriptionIds[0] != recipient.Id {
		t.Fatalf("expected gone subscription for %s, got %v", recipient.Id.Hex(), outcome.GoneSubscriptionIds)
	}
}

func TestDispatchTestSendsSmsToStaffOnly(t *testing.T) {
	sms := &fakeProvider{channel: constant.ChannelSms}
	push := &fakeProvider{channel: constant.ChannelPush}
	dispatcher := NewDispatcher(sms, push)
	staff := []entities.StaffPermission{
		{Id: primitive.NewObjectID(), Phone: "+66899998888"},
		{Id: primitive.NewObjectID(), Phone: "+66897776666"},
	}

	outcome := dispatcher.DispatchTest(sampleEvent(), staff, dispatchTemplate(), time.Hour)

	if len(outcome.Logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(outcome.Logs))
	}
	if outcome.Summary.Sms.Sent != 2 {
		t.Fatalf("expected 2 sms sent, got %+v", outcome.Summary)
	}
	if len(push.received) != 0 {
		t.Fatal("test alert must not hit customer push channel")
	}
}

func TestGroupByTextBatchesAndPreservesAll(t *testing.T) {
	messages := []OutboundMessage{
		{RecipientKey: "a", Text: "x"},
		{RecipientKey: "b", Text: "x"},
		{RecipientKey: "c", Text: "y"},
	}

	batches := groupByText(messages, 2)

	total := 0
	for _, batch := range batches {
		total += len(batch)
		text := batch[0].Text
		for _, message := range batch {
			if message.Text != text {
				t.Fatal("batch must contain a single text")
			}
		}
	}
	if total != 3 {
		t.Fatalf("expected 3 messages across batches, got %d", total)
	}
}
