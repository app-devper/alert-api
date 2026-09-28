package db

import (
	"context"

	"github.com/app-devper/um-api/servicekit/tenant"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

// Seeder prepares a tenant's database beyond its indexes.
type Seeder func(ctx context.Context, clientId string, database *mongo.Database) error

// Manager keeps one database per tenant through servicekit/tenant (um-api
// ADR-0007): <prefix>_<clientId>, created with its indexes and seed data on
// first use, and retried later if that fails.
type Manager struct {
	tenants *tenant.Registry[*mongo.Database]
}

func NewManager(client *mongo.Client, dbPrefix string, seeder Seeder) *Manager {
	open := func(name string) *mongo.Database { return client.Database(name) }
	initialise := func(ctx context.Context, clientID string, database *mongo.Database) error {
		if err := createTenantIndexes(ctx, database); err != nil {
			return err
		}
		if seeder != nil {
			if err := seeder(ctx, clientID, database); err != nil {
				return err
			}
		}
		logrus.Infof("Opened database %q for client %q", database.Name(), clientID)
		return nil
	}
	return &Manager{tenants: tenant.New(dbPrefix, open, initialise)}
}

// ValidateClientID refuses an empty or malformed client id.
func ValidateClientID(clientID string) error { return tenant.ValidateClientID(clientID) }

// DbNameFor is the tenant's database name.
func DbNameFor(dbPrefix string, clientID string) string { return tenant.DatabaseName(dbPrefix, clientID) }

func (m *Manager) ForClient(clientID string) (*mongo.Database, error) {
	return m.tenants.For(clientID)
}

func (m *Manager) CollectionFor(clientID string, name string) (*mongo.Collection, error) {
	database, err := m.ForClient(clientID)
	if err != nil {
		return nil, err
	}
	return database.Collection(name), nil
}

// KnownClients lists the tenants this process has opened.
func (m *Manager) KnownClients() []string { return m.tenants.Known() }

const (
	CollectionCheckIns         = "check_ins"
	CollectionOtpRequests      = "otp_requests"
	CollectionEmergencyEvents  = "emergency_events"
	CollectionMessageTemplates = "message_templates"
	CollectionDeliveryLogs     = "delivery_logs"
	CollectionAuditLogs        = "audit_logs"
	CollectionBranchSettings   = "branch_settings"
	CollectionQrTokens         = "qr_tokens"
	CollectionStaffPermissions = "staff_permissions"
	CollectionCounters         = "counters"
	CollectionMessagingConfigs = "messaging_configs"
)
