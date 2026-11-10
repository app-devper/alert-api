package repositories

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"alert/app/data/entities"
	"alert/db"

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

func TestCheckoutIgnoresACheckInOfAnotherBranch(t *testing.T) {
	checkIns := NewCheckInEntity(newTestResource(t))
	created, err := checkIns.CreateCheckIn(entities.CheckIn{ClientId: "c01", BranchId: "B1", Phone: "0812345678"})
	if err != nil {
		t.Fatal(err)
	}
	err = checkIns.Checkout("c01", "B2", created.Id, time.Now(), "STAFF:u1")
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("foreign Branch: got %v, want ErrNoDocuments", err)
	}
	got, err := checkIns.GetCheckInById("c01", created.Id)
	if err != nil || got.CheckedOutAt != nil {
		t.Fatalf("the B1 Check-in was checked out from B2 (err %v)", err)
	}
	if err := checkIns.Checkout("c01", "B1", created.Id, time.Now(), "STAFF:u1"); err != nil {
		t.Fatalf("own Branch: %v", err)
	}
	err = checkIns.Checkout("c01", "B1", created.Id, time.Now(), "STAFF:u1")
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("already checked out: got %v, want ErrNoDocuments", err)
	}
}

func TestGetEventByIdIgnoresAnEventOfAnotherBranch(t *testing.T) {
	events := NewEmergencyEventEntity(newTestResource(t))
	created, err := events.CreateEvent(entities.EmergencyEvent{ClientId: "c01", BranchId: "B1", EventType: "FIRE"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := events.GetEventById("c01", "B2", created.Id); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("foreign Branch: got %v, want ErrNoDocuments", err)
	}
	got, err := events.GetEventById("c01", "B1", created.Id)
	if err != nil || got.Id != created.Id {
		t.Fatalf("own Branch: got %v, %v", got.Id, err)
	}
}
