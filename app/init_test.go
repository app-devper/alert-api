package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"alert/app/core/config"
	"alert/app/domain"
	"alert/db"
	"alert/middlewares"

	"github.com/app-devper/um-api/sessionclient"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func testRepository(t *testing.T) *domain.Repository {
	t.Helper()
	mongoClient, err := mongo.NewClient(options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
	if err != nil {
		t.Fatalf("unexpected error building mongo client: %v", err)
	}
	resource := &db.Resource{
		Mongo: db.NewManager(mongoClient, "alert_init_test", nil),
		RdDb:  redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}),
	}
	cfg := &config.Config{
		Port:      "8089",
		MongoHost: "mongodb://127.0.0.1:27017",
		DbPrefix:  "alert_init_test",
		RedisHost: "redis://127.0.0.1:0",
		SecretKey: "test-secret-key",
		System:    "ALERT",
	}
	repository := domain.InitRepository(resource, cfg)
	auth, err := middlewares.NewAuthWithStore(cfg, unusedStore{})
	if err != nil {
		t.Fatal(err)
	}
	repository.Auth = auth
	return repository
}

// unusedStore stands in for UM's session store; anonymous requests never
// reach it.
type unusedStore struct{}

func (unusedStore) Session(context.Context, string) (sessionclient.Session, error) {
	return sessionclient.Session{}, sessionclient.ErrUnavailable
}

// publicPrefixes are the only /api/alert/v1 routes callable without a UM
// token: health, customer check-in (its own session), and provider webhooks.
var publicPrefixes = []string{"/api/alert/v1/health", "/api/alert/v1/public", "/api/alert/v1/webhook"}

func TestEveryStaffRouteRejectsAnAnonymousRequest(t *testing.T) {
	r := newTestEngine(t)
	checked := 0
	for _, route := range r.Routes() {
		if !strings.HasPrefix(route.Path, "/api/alert/v1") || hasAnyPrefix(route.Path, publicPrefixes) {
			continue
		}
		checked++
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(route.Method, strings.ReplaceAll(route.Path, ":", "x"), nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401 without a token, got %d", route.Method, route.Path, w.Code)
		}
	}
	if checked == 0 {
		t.Fatal("no staff routes were checked")
	}
	t.Logf("checked %d staff routes", checked)
}

func hasAnyPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func newTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, testRepository(t))
	return r
}

func TestRegisterRoutesExposesHealthCheck(t *testing.T) {
	r := newTestEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRegisterRoutesExposesVersionedHealthCheck(t *testing.T) {
	r := newTestEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/api/alert/v1/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRegisterRoutesRejectsUnauthenticatedStaffRequest(t *testing.T) {
	r := newTestEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/api/alert/v1/dashboard/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth header, got %d", w.Code)
	}
}

func TestRegisterRoutesUnknownPathReturns404(t *testing.T) {
	r := newTestEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
