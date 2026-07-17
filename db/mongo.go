package db

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

var validClientID = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9_-]{0,48}[a-zA-Z0-9])?$`)

type Seeder func(ctx context.Context, clientId string, database *mongo.Database) error

type tenantDB struct {
	db       *mongo.Database
	initOnce sync.Once
}

type Manager struct {
	client   *mongo.Client
	dbPrefix string
	seeder   Seeder
	cache    sync.Map
}

func NewManager(client *mongo.Client, dbPrefix string, seeder Seeder) *Manager {
	return &Manager{
		client:   client,
		dbPrefix: dbPrefix,
		seeder:   seeder,
	}
}

func ValidateClientID(clientID string) error {
	if clientID == "" {
		return errors.New("clientId is required")
	}
	if !validClientID.MatchString(clientID) {
		return fmt.Errorf("invalid clientId %q", clientID)
	}
	return nil
}

func DbNameFor(dbPrefix string, clientID string) string {
	if clientID == "000" {
		return dbPrefix
	}
	return fmt.Sprintf("%s_%s", dbPrefix, clientID)
}

func (m *Manager) ForClient(clientID string) (*mongo.Database, error) {
	if err := ValidateClientID(clientID); err != nil {
		return nil, err
	}
	if v, ok := m.cache.Load(clientID); ok {
		entry := v.(*tenantDB)
		m.ensureInitialized(entry, clientID)
		return entry.db, nil
	}
	entry := &tenantDB{db: m.client.Database(DbNameFor(m.dbPrefix, clientID))}
	actual, _ := m.cache.LoadOrStore(clientID, entry)
	entry = actual.(*tenantDB)
	m.ensureInitialized(entry, clientID)
	return entry.db, nil
}

func (m *Manager) CollectionFor(clientID string, name string) (*mongo.Collection, error) {
	database, err := m.ForClient(clientID)
	if err != nil {
		return nil, err
	}
	return database.Collection(name), nil
}

func (m *Manager) KnownClients() []string {
	clients := []string{}
	m.cache.Range(func(key, _ interface{}) bool {
		clients = append(clients, key.(string))
		return true
	})
	return clients
}

func (m *Manager) ensureInitialized(entry *tenantDB, clientID string) {
	entry.initOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := createTenantIndexes(ctx, entry.db); err != nil {
			logrus.Warnf("tenant %q: index creation reported error (continuing): %v", clientID, err)
		}
		if m.seeder != nil {
			if err := m.seeder(ctx, clientID, entry.db); err != nil {
				logrus.Warnf("tenant %q: seeder reported error (continuing): %v", clientID, err)
			}
		}
		logrus.Infof("Opened database %q for client %q", entry.db.Name(), clientID)
	})
}

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
)
