package middlewares

import (
	"errors"
	"net/http"

	"alert/app/core/config"
	"alert/app/core/errcode"
	"alert/app/data/repositories"
	"alert/app/domain/staff"
	"alert/db"

	"github.com/app-devper/um-api/sessionclient"
	"github.com/app-devper/um-api/sessionclient/ginauth"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
		if errors.Is(err, mongo.ErrNoDocuments) {
			errcode.Abort(ctx, http.StatusForbidden, errcode.AU_FORBIDDEN_002, "no Alert Permission")
			return
		}
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{"clientId": clientId, "userId": userId}).Warn("alert permission lookup failed")
			errcode.Abort(ctx, http.StatusServiceUnavailable, errcode.AU_UNAVAILABLE_002, "permission store unavailable")
			return
		}
		if !permission.Active {
			errcode.Abort(ctx, http.StatusForbidden, errcode.AU_FORBIDDEN_001, "staff permission inactive")
			return
		}
		if permission.BranchId == "" {
			errcode.Abort(ctx, http.StatusForbidden, errcode.AU_FORBIDDEN_002, "no Alert Permission")
			return
		}
		staff.Set(ctx, staff.FromPermission(userId, sessionclient.Role(ctx.GetString("Role")), permission))
		ctx.Next()
	}
}
