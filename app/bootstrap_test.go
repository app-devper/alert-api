package app

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"alert/app/core/config"
	"alert/app/domain"
	"alert/db"
	"alert/middlewares"

	"github.com/app-devper/um-api/sessionclient"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	bootstrapSecret   = "test-secret-key"
	bootstrapClientId = "c01"
)

// userSessions confirms any session id of the form "sid-<userId>".
type userSessions struct{}

func (userSessions) Session(_ context.Context, id string) (sessionclient.Session, error) {
	return sessionclient.Session{UserId: strings.TrimPrefix(id, "sid-"), System: "ALERT"}, nil
}

func adminToken(t *testing.T, userId string) string {
	t.Helper()
	claims := jwt.MapClaims{"role": "ADMIN", "system": "ALERT", "clientId": bootstrapClientId,
		"jti": "sid-" + userId, "exp": time.Now().Add(time.Hour).Unix()}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(bootstrapSecret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func bootstrapEngine(t *testing.T) *gin.Engine {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set")
	}
	gin.SetMode(gin.TestMode)
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("alertboot%d", time.Now().UnixNano()%1000000000)
	t.Cleanup(func() {
		_ = mongoClient.Database(db.DbNameFor(prefix, bootstrapClientId)).Drop(context.Background())
	})
	resource := &db.Resource{Mongo: db.NewManager(mongoClient, prefix, nil), RdDb: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}
	cfg := &config.Config{Port: "1", MongoHost: uri, DbPrefix: prefix, RedisHost: "redis://127.0.0.1:0", SecretKey: bootstrapSecret, System: "ALERT"}
	repository := domain.InitRepository(resource, cfg)
	if repository.Auth, err = middlewares.NewAuthWithStore(cfg, userSessions{}); err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	RegisterRoutes(r, repository)
	return r
}

func TestFirstAdminBootstrapsAndRecoversFromSelfLockout(t *testing.T) {
	r := bootstrapEngine(t)
	token := adminToken(t, "admin-1")
	call := func(method, path, body string) (int, string) {
		req := httptest.NewRequest(method, "/api/alert/v1"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code, w.Body.String()
	}
	expect := func(label string, wantStatus int, wantCode string, method, path, body string) {
		t.Helper()
		status, got := call(method, path, body)
		if status != wantStatus || !strings.Contains(got, wantCode) {
			t.Fatalf("%s: want %d containing %q, got %d %s", label, wantStatus, wantCode, status, got)
		}
	}
	grant := func(active bool) string {
		return fmt.Sprintf(`{"userId":"admin-1","branchId":"B1","active":%t}`, active)
	}

	expect("no Permission is refused", 403, "AU-403-003", "GET", "/admin/settings", "")
	expect("grant self a Branch", 200, "", "PUT", "/admin/permissions", grant(true))
	expect("Permission opens the route", 200, "", "GET", "/admin/settings", "")

	expect("deactivate self", 200, "", "PUT", "/admin/permissions", grant(false))
	expect("inactive Permission is refused", 403, "AU-403-001", "GET", "/admin/settings", "")
	expect("permissions list still answers", 200, "", "GET", "/admin/permissions", "")
	expect("reactivate self", 200, "", "PUT", "/admin/permissions", grant(true))
	expect("reactivated Permission opens the route", 200, "", "GET", "/admin/settings", "")
}
