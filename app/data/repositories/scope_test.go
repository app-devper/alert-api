package repositories

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"alert/app/core/constant"
	"alert/app/data/entities"
	"alert/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func newTestResource(t *testing.T) *db.Resource {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("alertscope%d", time.Now().UnixNano()%1000000000)
	t.Cleanup(func() {
		_ = client.Database(db.DbNameFor(prefix, "c01")).Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})
	return &db.Resource{Mongo: db.NewManager(client, prefix, nil)}
}

func TestAForeignBranchSeesNothing(t *testing.T) {
	res := newTestResource(t)
	checkIns, events := NewCheckInEntity(res), NewEmergencyEventEntity(res)
	own, foreign := Scope{ClientId: "c01", BranchId: "B1"}, Scope{ClientId: "c01", BranchId: "B2"}
	now := time.Now()
	// active per activeFilter: otpVerifiedAt set, checkedOutAt nil, expiresAt in the future
	verified, expires := now.Add(-time.Minute), now.Add(time.Hour)
	checkIn, err := checkIns.CreateCheckIn(entities.CheckIn{ClientId: "c01", BranchId: "B1", Phone: "0812345678", OtpVerifiedAt: &verified, ExpiresAt: expires})
	if err != nil {
		t.Fatal(err)
	}
	event, err := events.CreateEvent(entities.EmergencyEvent{ClientId: "c01", BranchId: "B1", EventType: "FIRE", Status: constant.EventStatusOpen, SentAt: now})
	if err != nil {
		t.Fatal(err)
	}

	// own Branch sees them
	if list, err := checkIns.GetActiveCheckIns(own, now); err != nil || len(list) != 1 {
		t.Fatalf("own Branch: %d active Check-ins (err %v), want 1", len(list), err)
	}
	if n, err := checkIns.CountActive(own, now); err != nil || n != 1 {
		t.Fatalf("own Branch CountActive: %d (err %v), want 1", n, err)
	}
	if latest, err := events.GetLatestEvent(own, "FIRE"); err != nil || latest == nil || latest.Id != event.Id {
		t.Fatalf("own Branch latest event: %+v, %v", latest, err)
	}
	if list, total, err := events.QueryEvents(EventQuery{Scope: own, Page: 1, Limit: 20}); err != nil || total != 1 || len(list) != 1 || list[0].Id != event.Id {
		t.Fatalf("own Branch QueryEvents: %+v total %d, %v", list, total, err)
	}
	if got, err := checkIns.GetCheckInById(own, checkIn.Id); err != nil || got.Id != checkIn.Id {
		t.Fatalf("own Branch Check-in: %v", err)
	}
	if got, err := events.GetEventById(own, event.Id); err != nil || got.Id != event.Id {
		t.Fatalf("own Branch event: %v", err)
	}
	if open, err := events.GetLatestOpenEvent(own); err != nil || open == nil || open.Id != event.Id {
		t.Fatalf("own Branch open event: %+v, %v", open, err)
	}

	// a foreign Branch sees nothing
	if _, err := checkIns.GetCheckInById(foreign, checkIn.Id); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("GetCheckInById: %v", err)
	}
	if list, err := checkIns.GetActiveCheckIns(foreign, now); err != nil || len(list) != 0 {
		t.Fatalf("GetActiveCheckIns: %d (err %v), want 0 and no error", len(list), err)
	}
	if n, err := checkIns.CountActive(foreign, now); err != nil || n != 0 {
		t.Fatalf("CountActive: %d (err %v), want 0 and no error", n, err)
	}
	if err := checkIns.Checkout(foreign, checkIn.Id, now, "STAFF:u1"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("Checkout: %v", err)
	}
	if got, err := checkIns.GetCheckInById(own, checkIn.Id); err != nil || got.CheckedOutAt != nil {
		t.Fatalf("the B1 Check-in was checked out from B2 (err %v)", err)
	}
	if _, err := events.GetEventById(foreign, event.Id); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("GetEventById: %v", err)
	}
	if latest, err := events.GetLatestEvent(foreign, "FIRE"); err != nil || latest != nil {
		t.Fatalf("GetLatestEvent: %+v, %v, want nil and no error", latest, err)
	}
	if open, err := events.GetLatestOpenEvent(foreign); err != nil || open != nil {
		t.Fatalf("GetLatestOpenEvent: %+v, %v, want nil and no error", open, err)
	}
	if list, total, err := events.QueryEvents(EventQuery{Scope: foreign, Page: 1, Limit: 20}); err != nil || total != 0 || len(list) != 0 {
		t.Fatalf("QueryEvents: %d events, total %d, %v, want none and no error", len(list), total, err)
	}
	if err := events.CloseOpenEvents(foreign, now); err != nil {
		t.Fatal(err)
	}
	if got, _ := events.GetEventById(own, event.Id); got.ClosedAt != nil || got.Status != constant.EventStatusOpen {
		t.Fatal("CloseOpenEvents from B2 closed a B1 event")
	}
}

func TestOwnBranchCheckoutSucceedsOnceThenReportsNoDocuments(t *testing.T) {
	checkIns := NewCheckInEntity(newTestResource(t))
	own := Scope{ClientId: "c01", BranchId: "B1"}
	created, err := checkIns.CreateCheckIn(entities.CheckIn{ClientId: "c01", BranchId: "B1", Phone: "0812345678"})
	if err != nil {
		t.Fatal(err)
	}
	if err := checkIns.Checkout(own, created.Id, time.Now(), "STAFF:u1"); err != nil {
		t.Fatalf("own Branch: %v", err)
	}
	got, err := checkIns.GetCheckInById(own, created.Id)
	if err != nil || got.CheckedOutAt == nil {
		t.Fatalf("Check-in not checked out (err %v)", err)
	}
	err = checkIns.Checkout(own, created.Id, time.Now(), "STAFF:u1")
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("already checked out: got %v, want ErrNoDocuments", err)
	}
}

func TestCloseOpenEventsClosesOwnBranchEvents(t *testing.T) {
	events := NewEmergencyEventEntity(newTestResource(t))
	own := Scope{ClientId: "c01", BranchId: "B1"}
	created, err := events.CreateEvent(entities.EmergencyEvent{ClientId: "c01", BranchId: "B1", EventType: "FIRE", Status: constant.EventStatusOpen, SentAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if err := events.CloseOpenEvents(own, time.Now()); err != nil {
		t.Fatal(err)
	}
	got, err := events.GetEventById(own, created.Id)
	if err != nil || got.ClosedAt == nil || got.Status != constant.EventStatusClosed {
		t.Fatalf("own Branch event not closed: %+v, %v", got, err)
	}
}

func TestFindCheckInByRefIsClientScoped(t *testing.T) {
	checkIns := NewCheckInEntity(newTestResource(t))
	created, err := checkIns.CreateCheckIn(entities.CheckIn{ClientId: "c01", BranchId: "B1", Phone: "0812345678"})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := checkIns.FindCheckInByRef("c01", created.Id); err != nil || got.BranchId != "B1" {
		t.Fatalf("got %v %v", got.BranchId, err)
	}
}

func TestScopeFilterAddsBranchWithoutMutatingItsArgument(t *testing.T) {
	fields := bson.M{"_id": 1}
	got := Scope{ClientId: "c01", BranchId: "B1"}.filter(fields)
	if got["branchId"] != "B1" || got["_id"] != 1 {
		t.Fatalf("filter: %v", got)
	}
	if _, mutated := fields["branchId"]; mutated {
		t.Fatal("filter mutated its argument")
	}
}

var (
	ownScope     = Scope{ClientId: "c01", BranchId: "B1"}
	foreignScope = Scope{ClientId: "c01", BranchId: "B2"}
)

func createB1CheckIn(t *testing.T, checkIns ICheckIn) entities.CheckIn {
	t.Helper()
	created, err := checkIns.CreateCheckIn(entities.CheckIn{ClientId: "c01", BranchId: "B1", Phone: "0812345678"})
	if err != nil {
		t.Fatal(err)
	}
	return created
}

func TestMarkOtpVerifiedOnlyTakesEffectInTheCheckInsBranch(t *testing.T) {
	checkIns := NewCheckInEntity(newTestResource(t))
	created := createB1CheckIn(t, checkIns)
	verifiedAt, expiresAt := time.Now().Truncate(time.Millisecond), time.Now().Add(time.Hour)

	if err := checkIns.MarkOtpVerified(foreignScope, created.Id, verifiedAt, "hash", expiresAt); err != nil {
		t.Fatalf("foreign Branch: %v", err)
	}
	got, err := checkIns.GetCheckInById(ownScope, created.Id)
	if err != nil || got.OtpVerifiedAt != nil || got.SessionTokenHash != "" {
		t.Fatalf("a foreign Branch verified the Check-in: %+v, %v", got, err)
	}

	if err := checkIns.MarkOtpVerified(ownScope, created.Id, verifiedAt, "hash", expiresAt); err != nil {
		t.Fatalf("own Branch: %v", err)
	}
	got, err = checkIns.GetCheckInById(ownScope, created.Id)
	if err != nil || got.OtpVerifiedAt == nil || got.SessionTokenHash != "hash" {
		t.Fatalf("own Branch did not verify the Check-in: %+v, %v", got, err)
	}
}

func TestPushSubscriptionOnlyChangesInTheCheckInsBranch(t *testing.T) {
	checkIns := NewCheckInEntity(newTestResource(t))
	created := createB1CheckIn(t, checkIns)
	subscription := &entities.PushSubscription{Endpoint: "https://push.example/1"}
	pushOf := func() *entities.PushSubscription {
		got, err := checkIns.GetCheckInById(ownScope, created.Id)
		if err != nil {
			t.Fatal(err)
		}
		return got.PushSubscription
	}

	if err := checkIns.SetPushSubscription(foreignScope, created.Id, subscription); err != nil {
		t.Fatalf("foreign set: %v", err)
	}
	if pushOf() != nil {
		t.Fatal("a foreign Branch set the push subscription")
	}
	if err := checkIns.SetPushSubscription(ownScope, created.Id, subscription); err != nil {
		t.Fatalf("own set: %v", err)
	}
	if got := pushOf(); got == nil || got.Endpoint != subscription.Endpoint {
		t.Fatalf("own Branch did not set the push subscription: %+v", got)
	}
	if err := checkIns.ClearPushSubscription(foreignScope, created.Id); err != nil {
		t.Fatalf("foreign clear: %v", err)
	}
	if pushOf() == nil {
		t.Fatal("a foreign Branch cleared the push subscription")
	}
	if err := checkIns.ClearPushSubscription(ownScope, created.Id); err != nil {
		t.Fatalf("own clear: %v", err)
	}
	if pushOf() != nil {
		t.Fatal("own Branch did not clear the push subscription")
	}
}

func TestDeleteCheckInOnlyTakesEffectInTheCheckInsBranch(t *testing.T) {
	checkIns := NewCheckInEntity(newTestResource(t))
	created := createB1CheckIn(t, checkIns)

	if err := checkIns.DeleteCheckIn(foreignScope, created.Id); err != nil {
		t.Fatalf("foreign Branch: %v", err)
	}
	if _, err := checkIns.GetCheckInById(ownScope, created.Id); err != nil {
		t.Fatalf("a foreign Branch deleted the Check-in: %v", err)
	}
	if err := checkIns.DeleteCheckIn(ownScope, created.Id); err != nil {
		t.Fatalf("own Branch: %v", err)
	}
	if _, err := checkIns.GetCheckInById(ownScope, created.Id); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("own Branch did not delete the Check-in: %v", err)
	}
}

func TestUpdateChannelSummaryOnlyTakesEffectInTheEventsBranch(t *testing.T) {
	events := NewEmergencyEventEntity(newTestResource(t))
	created, err := events.CreateEvent(entities.EmergencyEvent{ClientId: "c01", BranchId: "B1", EventType: "FIRE", Status: constant.EventStatusOpen, SentAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	summary := entities.ChannelSummary{Sms: entities.ChannelStat{Sent: 3, Delivered: 2, Failed: 1}}

	if err := events.UpdateChannelSummary(foreignScope, created.Id, summary, "ref-1"); err != nil {
		t.Fatalf("foreign Branch: %v", err)
	}
	got, err := events.GetEventById(ownScope, created.Id)
	if err != nil || got.ChannelSummary != (entities.ChannelSummary{}) || got.ProviderReference != "" {
		t.Fatalf("a foreign Branch updated the event: %+v, %v", got, err)
	}
	if err := events.UpdateChannelSummary(ownScope, created.Id, summary, "ref-1"); err != nil {
		t.Fatalf("own Branch: %v", err)
	}
	got, err = events.GetEventById(ownScope, created.Id)
	if err != nil || got.ChannelSummary != summary || got.ProviderReference != "ref-1" {
		t.Fatalf("own Branch did not update the event: %+v, %v", got, err)
	}
}
