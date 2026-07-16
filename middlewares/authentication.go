package middlewares

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"alert/app/core/constant"
	"alert/app/core/errcode"
	"alert/app/data/repositories"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/mongo"
)

type AccessClaims struct {
	Role     string `json:"role"`
	System   string `json:"system"`
	ClientId string `json:"clientId"`
	jwt.RegisteredClaims
}

type authConfig struct {
	jwtKey   []byte
	clientId string
	system   string
}

func RequireAuthenticated() gin.HandlerFunc {
	config, configErr := loadAuthConfig()
	return func(ctx *gin.Context) {
		if configErr != nil {
			errcode.Abort(ctx, http.StatusInternalServerError, errcode.SY_INTERNAL_001, configErr.Error())
			return
		}

		token := ctx.GetHeader("Authorization")
		if token == "" {
			errcode.Abort(ctx, http.StatusUnauthorized, errcode.AU_UNAUTHORIZED_001, "missing authorization header")
			return
		}
		jwtToken := strings.Split(token, "Bearer ")
		if len(jwtToken) < 2 {
			errcode.Abort(ctx, http.StatusUnauthorized, errcode.AU_UNAUTHORIZED_001, "missing authorization header")
			return
		}
		claims := &AccessClaims{}
		tkn, err := jwt.ParseWithClaims(jwtToken[1], claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return config.jwtKey, nil
		})
		if err != nil {
			errcode.Abort(ctx, http.StatusUnauthorized, errcode.AU_UNAUTHORIZED_002, err.Error())
			return
		}
		if tkn == nil || !tkn.Valid || claims.ID == "" {
			errcode.Abort(ctx, http.StatusUnauthorized, errcode.AU_UNAUTHORIZED_002, "token invalid")
			return
		}
		if config.system != claims.System {
			errcode.Abort(ctx, http.StatusUnauthorized, errcode.AU_UNAUTHORIZED_003, "system invalid")
			return
		}
		if config.clientId != claims.ClientId {
			errcode.Abort(ctx, http.StatusUnauthorized, errcode.AU_UNAUTHORIZED_004, "clientId invalid")
			return
		}

		ctx.Set("SessionId", claims.ID)
		ctx.Set("Role", claims.Role)
		ctx.Set("System", claims.System)
		ctx.Set("ClientId", claims.ClientId)
		ctx.Next()
	}
}

func RequireSession(sessionEntity repositories.ISession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sessionId := ctx.GetString("SessionId")
		userId, err := sessionEntity.GetSessionById(sessionId)
		if err != nil {
			errcode.Abort(ctx, http.StatusUnauthorized, errcode.AU_UNAUTHORIZED_005, "session invalid")
			return
		}
		ctx.Set("UserId", userId)
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

func loadAuthConfig() (*authConfig, error) {
	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		return nil, errors.New("missing required env: SECRET_KEY")
	}

	clientId := os.Getenv("CLIENT_ID")
	if clientId == "" {
		return nil, errors.New("missing required env: CLIENT_ID")
	}

	system := os.Getenv("SYSTEM")
	if system == "" {
		return nil, errors.New("missing required env: SYSTEM")
	}

	return &authConfig{
		jwtKey:   []byte(secretKey),
		clientId: clientId,
		system:   system,
	}, nil
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
	role := ctx.GetString("Role")
	if role == constant.SUPER || role == constant.ADMIN || role == constant.MANAGER {
		return true
	}
	for _, allowed := range AllowedEventTypes(ctx) {
		if allowed == eventType {
			return true
		}
	}
	return false
}
