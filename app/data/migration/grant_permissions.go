// Package migration holds one-off data repairs that ship with the service so
// they can be tested and re-run: here, the Alert Permissions that staff need
// now that a caller without one is refused (no "HQ" fallback).
package migration

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"alert/app/data/entities"
	"alert/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// RootClientID is the Client whose database carries the bare prefix.
const RootClientID = "000"

// customerActorPrefix marks audit rows written on behalf of a customer (a
// consent withdrawal), not a staff member.
const customerActorPrefix = "CUSTOMER:"

// Grant is one Alert Permission the repair would create: a staff member who
// has acted in a Client (audit log or Emergency Event) but holds no Permission.
type Grant struct {
	Database string
	ClientId string
	UserId   string
	BranchId string
	Source   string // "audit_logs", "emergency_events" or both
	LastSeen time.Time
}

// Tenant is one database the prefix owns and the Client it belongs to.
type Tenant struct {
	Database string
	ClientId string
}

// Planner finds the staff of every tenant database who need a Permission.
type Planner struct {
	Client *mongo.Client
	Prefix string
	// DefaultBranch is used when a staff member's activity names no Branch;
	// everything written under the old fallback lived at "HQ".
	DefaultBranch string
	// Clients, when set, limits the sweep to these clientIds.
	Clients []string
}

// Tenants lists the databases the prefix owns — the prefix itself (Client 000)
// and every "<prefix>_<clientId>" — plus the databases skipped because their
// derived clientId is not a valid one (another deployment sharing the cluster
// under a longer prefix, for example). The Clients filter is applied by
// Selected, not here, so the report can tell the two apart.
func (p Planner) Tenants(ctx context.Context) (tenants []Tenant, skipped []string, err error) {
	names, err := p.Client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		return nil, nil, fmt.Errorf("list databases: %w", err)
	}
	for _, name := range names {
		if name != p.Prefix && !strings.HasPrefix(name, p.Prefix+"_") {
			continue
		}
		clientID := clientIDOf(p.Prefix, name)
		if db.ValidateClientID(clientID) != nil {
			skipped = append(skipped, name)
			continue
		}
		tenants = append(tenants, Tenant{Database: name, ClientId: clientID})
	}
	sort.Slice(tenants, func(i, j int) bool { return tenants[i].Database < tenants[j].Database })
	sort.Strings(skipped)
	return tenants, skipped, nil
}

// Selected splits the tenants into those the Clients filter keeps and those it
// leaves out; with no filter every tenant is selected.
func (p Planner) Selected(tenants []Tenant) (selected, excluded []Tenant) {
	for _, t := range tenants {
		if p.wants(t.ClientId) {
			selected = append(selected, t)
		} else {
			excluded = append(excluded, t)
		}
	}
	return selected, excluded
}

func (p Planner) wants(clientID string) bool {
	if len(p.Clients) == 0 {
		return true
	}
	for _, c := range p.Clients {
		if c == clientID {
			return true
		}
	}
	return false
}

// Plan lists the Permissions missing across every tenant, without writing.
func (p Planner) Plan(ctx context.Context) ([]Grant, error) {
	tenants, _, err := p.Tenants(ctx)
	if err != nil {
		return nil, err
	}
	selected, _ := p.Selected(tenants)
	var grants []Grant
	for _, tenant := range selected {
		found, err := p.planTenant(ctx, tenant)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", tenant.Database, err)
		}
		grants = append(grants, found...)
	}
	return grants, nil
}

// Apply creates the Permissions Plan found: Active, no Event Types of their
// own (MANAGER and above never needed them; a USER without a Permission could
// not trigger before either), marked as written by `by`. A staff member who
// was granted a Permission between Plan and Apply is left as they are, so
// running Apply twice is safe. It reports how many Permissions it created.
func (p Planner) Apply(ctx context.Context, grants []Grant, by string) (int, error) {
	created := 0
	for _, g := range grants {
		permission := entities.StaffPermission{
			ClientId:          g.ClientId,
			UserId:            g.UserId,
			BranchId:          g.BranchId,
			AllowedEventTypes: []string{},
			Active:            true,
			UpdatedBy:         by,
			UpdatedAt:         time.Now(),
		}
		result, err := p.Client.Database(g.Database).Collection(db.CollectionStaffPermissions).UpdateOne(ctx,
			bson.M{"clientId": g.ClientId, "userId": g.UserId},
			bson.M{"$setOnInsert": permission},
			options.Update().SetUpsert(true))
		if err != nil {
			return created, fmt.Errorf("grant %s in %s: %w", g.UserId, g.Database, err)
		}
		if result.UpsertedCount > 0 {
			created++
		}
	}
	return created, nil
}

// planTenant merges the staff seen in the tenant's audit logs and Emergency
// Events, drops those who already hold a Permission, and fills in the Branch.
func (p Planner) planTenant(ctx context.Context, tenant Tenant) ([]Grant, error) {
	database := p.Client.Database(tenant.Database)
	seen := map[string]Grant{}

	audits, err := latestByUser(ctx, database.Collection(db.CollectionAuditLogs), tenant.ClientId, "actor", "occurredAt")
	if err != nil {
		return nil, fmt.Errorf("audit logs: %w", err)
	}
	for _, a := range audits {
		seen[a.User] = merged(seen[a.User], tenant, a, db.CollectionAuditLogs)
	}
	events, err := latestByUser(ctx, database.Collection(db.CollectionEmergencyEvents), tenant.ClientId, "triggeredBy", "sentAt")
	if err != nil {
		return nil, fmt.Errorf("emergency events: %w", err)
	}
	for _, e := range events {
		seen[e.User] = merged(seen[e.User], tenant, e, db.CollectionEmergencyEvents)
	}

	holders, err := database.Collection(db.CollectionStaffPermissions).Distinct(ctx, "userId", bson.M{})
	if err != nil {
		return nil, fmt.Errorf("existing permissions: %w", err)
	}
	for _, h := range holders {
		if userID, ok := h.(string); ok {
			delete(seen, userID)
		}
	}

	grants := make([]Grant, 0, len(seen))
	for _, g := range seen {
		if g.BranchId == "" {
			g = Grant{Database: g.Database, ClientId: g.ClientId, UserId: g.UserId, BranchId: p.DefaultBranch, Source: g.Source, LastSeen: g.LastSeen}
		}
		grants = append(grants, g)
	}
	sort.Slice(grants, func(i, j int) bool { return grants[i].UserId < grants[j].UserId })
	return grants, nil
}

// activity is one staff member's most recent trace in one collection.
type activity struct {
	User   string
	Branch string // the most recent non-empty Branch, or ""
	At     time.Time
}

// merged folds an activity into the Grant seen so far for that user: the more
// recent activity decides the Branch, but never replaces a known Branch with
// an empty one.
func merged(current Grant, tenant Tenant, a activity, source string) Grant {
	next := Grant{
		Database: tenant.Database,
		ClientId: tenant.ClientId,
		UserId:   a.User,
		BranchId: current.BranchId,
		LastSeen: current.LastSeen,
		Source:   current.Source,
	}
	if a.At.After(current.LastSeen) {
		next.LastSeen = a.At
		if a.Branch != "" {
			next.BranchId = a.Branch
		}
	}
	if next.BranchId == "" {
		next.BranchId = a.Branch
	}
	if !strings.Contains(current.Source, source) {
		if current.Source == "" {
			next.Source = source
		} else {
			next.Source = current.Source + "+" + source
		}
	}
	return next
}

// latestByUser reports, for every staff member named by userField in the
// collection, their most recent timeField and the most recent non-empty
// Branch they acted in. Rows written for customers are not staff.
func latestByUser(ctx context.Context, col *mongo.Collection, clientID, userField, timeField string) ([]activity, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"clientId": clientID,
			userField:  bson.M{"$nin": bson.A{"", nil}, "$not": bson.M{"$regex": "^" + customerActorPrefix}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: timeField, Value: -1}}}},
		{{Key: "$group", Value: bson.M{
			"_id":      "$" + userField,
			"at":       bson.M{"$first": "$" + timeField},
			"branches": bson.M{"$push": "$branchId"},
		}}},
	}
	cursor, err := col.Aggregate(ctx, pipeline, options.Aggregate().SetAllowDiskUse(true))
	if err != nil {
		return nil, fmt.Errorf("aggregate %s by %s: %w", col.Name(), userField, err)
	}
	defer cursor.Close(ctx)
	var found []activity
	for cursor.Next(ctx) {
		var row struct {
			User     string    `bson:"_id"`
			At       time.Time `bson:"at"`
			Branches []string  `bson:"branches"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, fmt.Errorf("decode %s row: %w", col.Name(), err)
		}
		found = append(found, activity{User: row.User, Branch: firstNonEmpty(row.Branches), At: row.At})
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("read %s rows: %w", col.Name(), err)
	}
	return found, nil
}

func firstNonEmpty(values []string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// clientIDOf reverses db.DbNameFor: the bare prefix is Client 000, anything
// else is "<prefix>_<clientId>".
func clientIDOf(prefix, database string) string {
	if database == prefix {
		return RootClientID
	}
	return strings.TrimPrefix(database, prefix+"_")
}
