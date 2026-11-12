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
	if list, _ := checkIns.GetActiveCheckIns(own, now); len(list) != 1 {
		t.Fatalf("own Branch: %d active Check-ins, want 1", len(list))
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
	if list, _ := checkIns.GetActiveCheckIns(foreign, now); len(list) != 0 {
		t.Fatalf("GetActiveCheckIns: %d, want 0", len(list))
	}
	if n, _ := checkIns.CountActive(foreign, now); n != 0 {
		t.Fatalf("CountActive: %d, want 0", n)
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
	if latest, _ := events.GetLatestEvent(foreign, "FIRE"); latest != nil {
		t.Fatalf("GetLatestEvent: %+v, want nil", latest)
	}
	if open, _ := events.GetLatestOpenEvent(foreign); open != nil {
		t.Fatalf("GetLatestOpenEvent: %+v, want nil", open)
	}
	if _, total, _ := events.QueryEvents(EventQuery{Scope: foreign, Page: 1, Limit: 20}); total != 0 {
		t.Fatalf("QueryEvents: %d, want 0", total)
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
