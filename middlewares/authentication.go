package middlewares

import (
	"errors"
	"net/http"

	"alert/app/core/config"
	"alert/app/core/errcode"
	"alert/app/data/repositories"
	"alert/db"

	"github.com/app-devper/um-api/sessionclient"
	"github.com/app-devper/um-api/sessionclient/ginauth"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// NewAuth verifies UM access tokens for alert: SYSTEM binds the token,
// CLIENT_ID (optional) pins its client, and the session is confirmed in UM's
// Redis at REDIS_HOST (um-api ADR-0005). It fails when any required value is
// missing.
func NewAuth(cfg *config.Config) (*ginauth.Auth, error) {
	store, err := sessionclient.RedisStoreFor(cfg.RedisHost)
	if err != nil {
		return nil, err
	}
	return NewAuthWithStore(cfg, store)
}

// NewAuthWithStore is NewAuth with UM's session store supplied, for tests.
func NewAuthWithStore(cfg *config.Config, store sessionclient.Store) (*ginauth.Auth, error) {
	verifier, err := sessionclient.NewVerifier(sessionclient.Config{
		SecretKey: cfg.SecretKey,
		System:    cfg.System,
		ClientID:  cfg.ClientId,
		Store:     store,
	})
	if err != nil {
		return nil, err
	}
	return ginauth.New(verifier, func(ctx *gin.Context, e *sessionclient.Error) {
		errcode.Abort(ctx, e.Status, e.Code, e.Message)
	}), nil
}

// RequireSession admits a caller with a live UM session. Every alert route
// uses the default outage policy: while UM is unreachable a read may continue
// with the session last confirmed for its token, and writes wait.
func RequireSession(auth *ginauth.Auth) gin.HandlerFunc {
	return auth.Require(sessionclient.ReadOnlyWithLastGood)
}

// RequireTenant refuses a clientId that cannot name a tenant database. It
// runs after RequireSession.
func RequireTenant() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := db.ValidateClientID(ctx.GetString("ClientId")); err != nil {
			errcode.Abort(ctx, http.StatusUnauthorized, errcode.AU_UNAUTHORIZED_004, "clientId invalid")
			return
		}
		ctx.Next()
	}
}

func RequireBranch(permissionEntity repositories.IStaffPermission) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientId := ctx.GetString("ClientId")
		userId := ctx.GetString("UserId")
		permission, err := permissionEntity.GetByUserId(clientId, userId)
		if err != nil {
			if !errors.Is(err, mongo.ErrNoDocuments) {
				errcode.Abort(ctx, http.StatusForbidden, errcode.AU_FORBIDDEN_001, "permission lookup failed")
				return
			}
			ctx.Set("BranchId", fallbackBranch(ctx))
			ctx.Set("AllowedEventTypes", []string{})
			ctx.Next()
			return
		}
		if !permission.Active {
			errcode.Abort(ctx, http.StatusForbidden, errcode.AU_FORBIDDEN_001, "staff permission inactive")
			return
		}
		ctx.Set("BranchId", permission.BranchId)
		ctx.Set("AllowedEventTypes", permission.AllowedEventTypes)
		ctx.Next()
	}
}

func fallbackBranch(ctx *gin.Context) string {
	if branchId := ctx.GetHeader("X-Branch-Id"); branchId != "" {
		return branchId
	}
	return "HQ"
}

func AllowedEventTypes(ctx *gin.Context) []string {
	value, exists := ctx.Get("AllowedEventTypes")
	if !exists {
		return nil
	}
	allowed, ok := value.([]string)
	if !ok {
		return nil
	}
	return allowed
}

func CanTriggerEventType(ctx *gin.Context, eventType string) bool {
	if sessionclient.Role(ctx.GetString("Role")).AtLeast(sessionclient.RoleManager) {
		return true
	}
	for _, allowed := range AllowedEventTypes(ctx) {
		if allowed == eventType {
			return true
		}
	}
	return false
}
