package checkin

import (
	"net/http"
	"os"
	"time"

	"alert/app/core/alerting"
	"alert/app/core/constant"
	"alert/app/core/errcode"
	"alert/app/core/response"
	"alert/app/data/entities"
	"alert/app/domain"
	"alert/app/domain/request"
	"alert/db"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ApplyCheckInAPI(route *gin.RouterGroup, repository *domain.Repository) {
	r := route.Group("public")

	r.GET("/qr/:token", func(ctx *gin.Context) {
		handleResolveQr(ctx, repository)
	})

	r.GET("/branch", func(ctx *gin.Context) {
		handleResolveBranch(ctx, repository)
	})

	r.POST("/check-ins", func(ctx *gin.Context) {
		handleCreateCheckIn(ctx, repository)
	})

	r.POST("/check-ins/:id/verify", func(ctx *gin.Context) {
		handleVerifyOtp(ctx, repository)
	})

	r.POST("/check-ins/:id/resend-otp", func(ctx *gin.Context) {
		handleResendOtp(ctx, repository)
	})

	r.GET("/vapid", func(ctx *gin.Context) {
		response.Ok(ctx, gin.H{"publicKey": os.Getenv("VAPID_PUBLIC_KEY")})
	})

	me := r.Group("/me", requireCustomerSession(repository))
	me.GET("", func(ctx *gin.Context) {
		handleMe(ctx, repository)
	})
	me.POST("/checkout", func(ctx *gin.Context) {
		handleSelfCheckout(ctx, repository)
	})
	me.POST("/push", func(ctx *gin.Context) {
		handlePushSubscribe(ctx, repository)
	})
	me.DELETE("/push", func(ctx *gin.Context) {
		handlePushUnsubscribe(ctx, repository)
	})
	me.POST("/withdraw", func(ctx *gin.Context) {
		handleWithdrawConsent(ctx, repository)
	})
}

func handleResolveQr(ctx *gin.Context, repository *domain.Repository) {
	token := ctx.Param("token")
	clientId, _, err := alerting.SplitTenantRef(token)
	if err != nil {
		errcode.Abort(ctx, http.StatusGone, errcode.CK_GONE_001, "QR code is no longer valid")
		return
	}
	qrToken, err := repository.QrToken.GetActiveByToken(clientId, token)
	if err != nil {
		errcode.Abort(ctx, http.StatusGone, errcode.CK_GONE_001, "QR code is no longer valid")
		return
	}
	respondBranchInfo(ctx, repository, qrToken.ClientId, qrToken.BranchId, qrToken.TableNo)
}

func handleResolveBranch(ctx *gin.Context, repository *domain.Repository) {
	clientId := ctx.Query("clientId")
	branchId := ctx.Query("branchId")
	if err := db.ValidateClientID(clientId); err != nil || branchId == "" {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.CK_BAD_REQUEST_001, "clientId and branchId are required")
		return
	}
	respondBranchInfo(ctx, repository, clientId, branchId, ctx.Query("tableNo"))
}

func respondBranchInfo(ctx *gin.Context, repository *domain.Repository, clientId string, branchId string, tableNo string) {
	setting, err := repository.BranchSetting.GetSetting(clientId, branchId)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.CK_INTERNAL_001, err.Error())
		return
	}
	response.Ok(ctx, gin.H{
		"clientId":       clientId,
		"branchId":       branchId,
		"tableNo":        tableNo,
		"shopName":       setting.ShopName,
		"retentionHours": setting.RetentionHours,
		"contactChannel": setting.ContactChannel,
		"skipOtp":        setting.SkipOtp,
	})
}

func requireCustomerSession(repository *domain.Repository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("X-Session-Token")
		if token == "" {
			errcode.Abort(ctx, http.StatusUnauthorized, errcode.CK_UNAUTHORIZED_001, "missing session token")
			return
		}
		clientId, _, err := alerting.SplitTenantRef(token)
		if err != nil {
			errcode.Abort(ctx, http.StatusUnauthorized, errcode.CK_UNAUTHORIZED_001, "invalid session token")
			return
		}
		checkIn, err := repository.CheckIn.GetCheckInBySessionTokenHash(clientId, alerting.HashToken(token))
		if err != nil {
			errcode.Abort(ctx, http.StatusUnauthorized, errcode.CK_UNAUTHORIZED_001, "invalid session token")
			return
		}
		ctx.Set("CheckIn", checkIn)
		ctx.Next()
	}
}

func currentCheckIn(ctx *gin.Context) entities.CheckIn {
	value, _ := ctx.Get("CheckIn")
	checkIn, _ := value.(entities.CheckIn)
	return checkIn
}

func handleMe(ctx *gin.Context, repository *domain.Repository) {
	checkIn := currentCheckIn(ctx)
	status := "ACTIVE"
	if checkIn.CheckedOutAt != nil {
		status = "CHECKED_OUT"
	} else if !checkIn.ExpiresAt.After(time.Now()) {
		status = "EXPIRED"
	}
	channelConfig := repository.ProviderConfigFor(checkIn.ClientId)
	response.Ok(ctx, gin.H{
		"checkInNo":         checkIn.CheckInNo,
		"branchId":          checkIn.BranchId,
		"tableNo":           checkIn.TableNo,
		"groupSize":         checkIn.GroupSize,
		"phoneMasked":       alerting.MaskPhoneDisplay(checkIn.Phone),
		"preferredLanguage": checkIn.PreferredLanguage,
		"marketingConsent":  checkIn.MarketingConsent,
		"checkedInAt":       checkIn.CheckedInAt,
		"expiresAt":         checkIn.ExpiresAt,
		"status":            status,
		"channels": gin.H{
			"sms":  channelConfig.SmsEnabled,
			"push": checkIn.HasPush(),
			"line": channelConfig.LineEnabled,
		},
		"consent": gin.H{
			"consentAt":            checkIn.ConsentAt,
			"privacyNoticeVersion": checkIn.PrivacyNoticeVersion,
		},
	})
}

func handleSelfCheckout(ctx *gin.Context, repository *domain.Repository) {
	checkIn := currentCheckIn(ctx)
	if err := repository.CheckIn.Checkout(checkIn.ClientId, checkIn.Id, time.Now(), constant.CheckedOutBySelf); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.CK_INTERNAL_001, err.Error())
		return
	}
	response.Ok(ctx, gin.H{"status": "CHECKED_OUT"})
}

func handlePushSubscribe(ctx *gin.Context, repository *domain.Repository) {
	var req request.PushSubscribe
	if err := ctx.ShouldBindJSON(&req); err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.CK_BAD_REQUEST_001, err.Error())
		return
	}
	checkIn := currentCheckIn(ctx)
	subscription := &entities.PushSubscription{
		Endpoint: req.Endpoint,
		Keys:     entities.PushKeys{P256dh: req.P256dh, Auth: req.Auth},
	}
	if err := repository.CheckIn.SetPushSubscription(checkIn.ClientId, checkIn.Id, subscription); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.CK_INTERNAL_001, err.Error())
		return
	}
	response.Ok(ctx, gin.H{"push": true})
}

func handlePushUnsubscribe(ctx *gin.Context, repository *domain.Repository) {
	checkIn := currentCheckIn(ctx)
	if err := repository.CheckIn.ClearPushSubscription(checkIn.ClientId, checkIn.Id); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.CK_INTERNAL_001, err.Error())
		return
	}
	response.Ok(ctx, gin.H{"push": false})
}

func handleWithdrawConsent(ctx *gin.Context, repository *domain.Repository) {
	checkIn := currentCheckIn(ctx)
	if err := repository.CheckIn.DeleteCheckIn(checkIn.ClientId, checkIn.Id); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.CK_INTERNAL_001, err.Error())
		return
	}
	repository.AuditLog.Record(entities.AuditLog{
		ClientId: checkIn.ClientId,
		BranchId: checkIn.BranchId,
		Actor:    "CUSTOMER:" + checkIn.CheckInNo,
		Action:   constant.ActionWithdrawConsent,
		Result:   constant.ResultSuccess,
	})
	response.Ok(ctx, gin.H{"deleted": true})
}

func parseCheckInRef(ctx *gin.Context) (string, primitive.ObjectID, bool) {
	clientId, hex, err := alerting.SplitTenantRef(ctx.Param("id"))
	if err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.CK_BAD_REQUEST_001, "invalid id")
		return "", primitive.NilObjectID, false
	}
	id, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.CK_BAD_REQUEST_001, "invalid id")
		return "", primitive.NilObjectID, false
	}
	return clientId, id, true
}
