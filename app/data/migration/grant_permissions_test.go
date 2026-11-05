package migration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"alert/app/data/entities"
	"alert/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Two tenants: Client 000 on the bare prefix and Client 001. Staff a1, u1, m1
// and o1 have acted and hold no Permission; p1 already has one; a customer's
// consent withdrawal is audited as CUSTOMER:… and is not staff; a database
// named "<prefix>_" is not a tenant.
type fixture struct {
	Planner
	ctx  context.Context
	root *mongo.Database
	now  time.Time
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("alertgrant%d", time.Now().UnixNano())
	root := client.Database(db.DbNameFor(prefix, RootClientID))
	other := client.Database(db.DbNameFor(prefix, "001"))
	stray := client.Database(prefix + "_")
	t.Cleanup(func() {
		for _, d := range []*mongo.Database{root, other, stray} {
			_ = d.Drop(ctx)
		}
		_ = client.Disconnect(ctx)
	})
	now := time.Now().Truncate(time.Second)
	insert := func(d *mongo.Database, col string, docs ...any) {
		if _, err := d.Collection(col).InsertMany(ctx, docs); err != nil {
			t.Fatal(err)
		}
	}
	insert(root, db.CollectionAuditLogs,
		bson.M{"clientId": "000", "branchId": "HQ", "actor": "a1", "action": "x", "occurredAt": now.Add(-2 * time.Hour)},
		bson.M{"clientId": "000", "branchId": "B2", "actor": "a1", "action": "x", "occurredAt": now.Add(-1 * time.Hour)},
		bson.M{"clientId": "000", "branchId": "HQ", "actor": "p1", "action": "x", "occurredAt": now},
		bson.M{"clientId": "000", "branchId": "B1", "actor": "u1", "action": "x", "occurredAt": now.Add(-2 * time.Hour)},
		bson.M{"clientId": "000", "branchId": "", "actor": "u1", "action": "x", "occurredAt": now},
		bson.M{"clientId": "000", "branchId": "", "actor": "n1", "action": "x", "occurredAt": now},
		bson.M{"clientId": "000", "branchId": "HQ", "actor": "", "action": "x", "occurredAt": now},
		bson.M{"clientId": "000", "branchId": "HQ", "actor": "CUSTOMER:CK-000123", "action": "withdraw", "occurredAt": now},
	)
	insert(root, db.CollectionEmergencyEvents,
		bson.M{"clientId": "000", "branchId": "HQ", "triggeredBy": "m1", "eventType": "FIRE", "sentAt": now},
		bson.M{"clientId": "000", "branchId": "B3", "triggeredBy": "a1", "eventType": "FIRE", "sentAt": now.Add(-3 * time.Hour)},
	)
	insert(root, db.CollectionStaffPermissions,
		entities.StaffPermission{ClientId: "000", UserId: "p1", BranchId: "HQ", Active: true},
	)
	insert(other, db.CollectionAuditLogs,
		bson.M{"clientId": "001", "branchId": "S1", "actor": "o1", "action": "x", "occurredAt": now},
	)
	insert(stray, db.CollectionAuditLogs,
		bson.M{"clientId": "", "branchId": "X", "actor": "x1", "action": "x", "occurredAt": now},
	)
	return fixture{Planner: Planner{Client: client, Prefix: prefix, DefaultBranch: "HQ"}, ctx: ctx, root: root, now: now}
}

func (f fixture) grantsByKey(t *testing.T) map[string]Grant {
	t.Helper()
	grants, err := f.Plan(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Grant{}
	for _, g := range grants {
		got[g.Database+"/"+g.UserId] = g
	}
	return got
}

func TestTenantsAreThePrefixAndItsClients_NotAStrayDatabase(t *testing.T) {
	f := newFixture(t)
	tenants, skipped, err := f.Tenants(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(tenants) != 2 || tenants[0].ClientId != RootClientID || tenants[1].ClientId != "001" {
		t.Fatalf("tenants = %+v", tenants)
	}
	if len(skipped) != 1 || skipped[0] != f.Prefix+"_" {
		t.Fatalf("skipped = %v, want the database whose clientId is empty", skipped)
	}
}

func TestClientsFlagLimitsTheSweep(t *testing.T) {
	f := newFixture(t)
	f.Clients = []string{"001"}
	got := f.grantsByKey(t)
	if len(got) != 1 {
		t.Fatalf("got %v, want only Client 001's o1", got)
	}
}

func TestPlanListsStaffWithoutAPermissionAndTheirLatestBranch(t *testing.T) {
	f := newFixture(t)
	got := f.grantsByKey(t)
	root := db.DbNameFor(f.Prefix, RootClientID)
	want := map[string]string{
		root + "/a1":                          "B2", // audit log an hour ago beats the event three hours ago
		root + "/u1":                          "B1", // the latest row names no Branch; the older one does
		root + "/n1":                          "HQ", // no row ever named a Branch: the default
		root + "/m1":                          "HQ", // only an Emergency Event
		db.DbNameFor(f.Prefix, "001") + "/o1": "S1",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d grants %v, want %d", len(got), got, len(want))
	}
	for key, branch := range want {
		g, ok := got[key]
		if !ok || g.BranchId != branch {
			t.Fatalf("%s: got %+v, want Branch %s", key, g, branch)
		}
	}
	if _, ok := got[root+"/p1"]; ok {
		t.Fatal("p1 already holds a Permission and must not be granted again")
	}
	if _, ok := got[root+"/CUSTOMER:CK-000123"]; ok {
		t.Fatal("a customer's consent withdrawal is not a staff member")
	}
	if got[root+"/a1"].Source != "audit_logs+emergency_events" {
		t.Fatalf("a1 source = %q", got[root+"/a1"].Source)
	}
	if got[db.DbNameFor(f.Prefix, "001")+"/o1"].ClientId != "001" {
		t.Fatalf("o1 clientId = %q", got[db.DbNameFor(f.Prefix, "001")+"/o1"].ClientId)
	}
}

func TestApplyCreatesActivePermissionsOnce_SkipsOnesGrantedMeanwhile(t *testing.T) {
	f := newFixture(t)
	grants, err := f.Plan(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	// An admin grants m1 through the UI between the dry run and --apply.
	if _, err := f.root.Collection(db.CollectionStaffPermissions).InsertOne(f.ctx,
		entities.StaffPermission{ClientId: "000", UserId: "m1", BranchId: "B9", Active: true, UpdatedBy: "admin"}); err != nil {
		t.Fatal(err)
	}

	created, err := f.Apply(f.ctx, grants, "migration:test")
	if err != nil {
		t.Fatal(err)
	}
	if created != len(grants)-1 {
		t.Fatalf("created %d of %d, want all but the one granted meanwhile", created, len(grants))
	}

	var perm entities.StaffPermission
	if err := f.root.Collection(db.CollectionStaffPermissions).FindOne(f.ctx, bson.M{"userId": "a1"}).Decode(&perm); err != nil {
		t.Fatal(err)
	}
	if !perm.Active || perm.BranchId != "B2" || perm.ClientId != "000" || perm.UpdatedBy != "migration:test" || len(perm.AllowedEventTypes) != 0 {
		t.Fatalf("a1 permission = %+v", perm)
	}
	if err := f.root.Collection(db.CollectionStaffPermissions).FindOne(f.ctx, bson.M{"userId": "m1"}).Decode(&perm); err != nil {
		t.Fatal(err)
	}
	if perm.BranchId != "B9" || perm.UpdatedBy != "admin" {
		t.Fatalf("the admin's grant for m1 was overwritten: %+v", perm)
	}

	again, err := f.Apply(f.ctx, grants, "migration:test")
	if err != nil {
		t.Fatal(err)
	}
	if again != 0 {
		t.Fatalf("a second Apply created %d", again)
	}
	if left := f.grantsByKey(t); len(left) != 0 {
		t.Fatalf("second plan still lists %v", left)
	}
}
