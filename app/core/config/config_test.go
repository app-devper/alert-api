package config

import "testing"

func TestMustLoadReadsRequiredEnv(t *testing.T) {
	t.Setenv("PORT", "8089")
	t.Setenv("MONGO_HOST", "mongodb://localhost:27017")
	t.Setenv("REDIS_HOST", "redis://localhost:6379")
	t.Setenv("SECRET_KEY", "test-secret")
	t.Setenv("SYSTEM", "ALERT")
	t.Setenv("CLIENT_ID", "")
	t.Setenv("MONGO_DB_PREFIX", "")

	cfg := MustLoad()

	if cfg.Port != "8089" || cfg.MongoHost != "mongodb://localhost:27017" || cfg.RedisHost != "redis://localhost:6379" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.SecretKey != "test-secret" || cfg.System != "ALERT" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestMustLoadDefaultsDbPrefixWhenUnset(t *testing.T) {
	t.Setenv("PORT", "8089")
	t.Setenv("MONGO_HOST", "mongodb://localhost:27017")
	t.Setenv("REDIS_HOST", "redis://localhost:6379")
	t.Setenv("SECRET_KEY", "test-secret")
	t.Setenv("SYSTEM", "ALERT")
	t.Setenv("MONGO_DB_PREFIX", "")

	cfg := MustLoad()

	if cfg.DbPrefix != "alert" {
		t.Fatalf("expected default db prefix alert, got %q", cfg.DbPrefix)
	}
}

func TestMustLoadKeepsExplicitDbPrefix(t *testing.T) {
	t.Setenv("PORT", "8089")
	t.Setenv("MONGO_HOST", "mongodb://localhost:27017")
	t.Setenv("REDIS_HOST", "redis://localhost:6379")
	t.Setenv("SECRET_KEY", "test-secret")
	t.Setenv("SYSTEM", "ALERT")
	t.Setenv("MONGO_DB_PREFIX", "alert_002")

	cfg := MustLoad()

	if cfg.DbPrefix != "alert_002" {
		t.Fatalf("expected alert_002, got %q", cfg.DbPrefix)
	}
}

func TestMustLoadLeavesClientIdOptional(t *testing.T) {
	t.Setenv("PORT", "8089")
	t.Setenv("MONGO_HOST", "mongodb://localhost:27017")
	t.Setenv("REDIS_HOST", "redis://localhost:6379")
	t.Setenv("SECRET_KEY", "test-secret")
	t.Setenv("SYSTEM", "ALERT")
	t.Setenv("CLIENT_ID", "")

	cfg := MustLoad()

	if cfg.ClientId != "" {
		t.Fatalf("expected empty ClientId by default, got %q", cfg.ClientId)
	}
}
