package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"alert/app/core/config"
	"alert/app/domain"
	"alert/db"

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
	return domain.InitRepository(resource, cfg)
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
