package checkin

import (
	"fmt"
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

	"github.com/gin-gonic/gin"
)

const otpPerPhoneLimit = 3
const otpPerPhoneWindow = 10 * time.Minute
const otpPerClientLimit = 300
const otpPerClientWindow = time.Hour
const pendingCheckInTtl = 30 * time.Minute

func handleCreateCheckIn(ctx *gin.Context, repository *domain.Repository) {
	var req request.CreateCheckIn
	if err := ctx.ShouldBindJSON(&req); err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.CK_BAD_REQUEST_001, err.Error())
		return
	}
	if !req.AcceptPrivacyNotice {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.CK_BAD_REQUEST_002, "privacy notice must be accepted")
		return
	}
	phone, err := alerting.NormalizeThaiPhone(req.Phone)
	if err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.CK_BAD_REQUEST_002, "invalid phone number")
		return
	}
	clientId, _, err := alerting.SplitTenantRef(req.QrToken)
	if err != nil {
		errcode.Abort(ctx, http.StatusGone, errcode.CK_GONE_001, "QR code is no longer valid")
		return
	}
	qrToken, err := repository.QrToken.GetActiveByToken(clientId, req.QrToken)
	if err != nil {
		errcode.Abort(ctx, http.StatusGone, errcode.CK_GONE_001, "QR code is no longer valid")
		return
	}

	now := time.Now()
	sequence, err := repository.Counter.NextSequence(qrToken.ClientId, "CI", now)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.CK_INTERNAL_001, err.Error())
		return
	}
	tableNo := req.TableNo
	if tableNo == "" {
		tableNo = qrToken.TableNo
	}
	checkIn, err := repository.CheckIn.CreateCheckIn(entities.CheckIn{
		CheckInNo:            alerting.FormatCheckInNo(now, sequence),
		ClientId:             qrToken.ClientId,
		BranchId:             qrToken.BranchId,
		Phone:                phone,
		GroupSize:            req.GroupSize,
		TableNo:              tableNo,
		PreferredLanguage:    alerting.NormalizeLanguage(req.PreferredLanguage),
		MarketingConsent:     req.MarketingConsent,
		ConsentAt:            now,
		PrivacyNoticeVersion: req.PrivacyNoticeVersion,
		CheckedInAt:          now,
		ExpiresAt:            now.Add(pendingCheckInTtl),
	})
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.CK_INTERNAL_001, err.Error())
		return
	}
	sendOtp(ctx, repository, checkIn)
}

func handleResendOtp(ctx *gin.Context, repository *domain.Repository) {
	clientId, id, ok := parseCheckInRef(ctx)
	if !ok {
		return
	}
	checkIn, err := repository.CheckIn.GetCheckInById(clientId, id)
	if err != nil {
		errcode.Abort(ctx, http.StatusNotFound, errcode.CK_NOT_FOUND_001, "check-in not found")
		return
	}
	if checkIn.OtpVerifiedAt != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.OT_BAD_REQUEST_002, "already verified")
		return
	}
	sendOtp(ctx, repository, checkIn)
}

func sendOtp(ctx *gin.Context, repository *domain.Repository, checkIn entities.CheckIn) {
	if !passOtpRateLimits(ctx, repository, checkIn) {
		return
	}
	otp, err := alerting.GenerateOtp()
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.OT_INTERNAL_001, err.Error())
		return
	}
	refCode, err := alerting.GenerateRefCode()
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.OT_INTERNAL_001, err.Error())
		return
	}
	expiresAt := time.Now().Add(constant.OtpExpiryMinutes * time.Minute)
	_, err = repository.OtpRequest.CreateOtpRequest(entities.OtpRequest{
		ClientId:  checkIn.ClientId,
		CheckInId: checkIn.Id,
		Phone:     checkIn.Phone,
		OtpHash:   alerting.HashOtp(otpSecret(), checkIn.Phone, refCode, otp),
		RefCode:   refCode,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.OT_INTERNAL_001, err.Error())
		return
	}
	if err := repository.OtpSender.SendOtp(repository.ProviderConfigFor(checkIn.ClientId), checkIn.Phone, refCode, otp); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.OT_INTERNAL_001, "failed to send OTP")
		return
	}
	response.Ok(ctx, gin.H{
		"checkInId":    alerting.ComposeTenantRef(checkIn.ClientId, checkIn.Id.Hex()),
		"refCode":      refCode,
		"otpExpiresAt": expiresAt,
	})
}

func passOtpRateLimits(ctx *gin.Context, repository *domain.Repository, checkIn entities.CheckIn) bool {
	phoneKey := fmt.Sprintf("otp:phone:%s:%s", checkIn.ClientId, checkIn.Phone)
	phoneCount, err := repository.RateLimit.Increment(phoneKey, otpPerPhoneWindow)
	if err == nil && phoneCount > otpPerPhoneLimit {
		errcode.Abort(ctx, http.StatusTooManyRequests, errcode.OT_TOO_MANY_001, "too many OTP requests for this number")
		return false
	}
	clientKey := fmt.Sprintf("otp:client:%s", checkIn.ClientId)
	clientCount, err := repository.RateLimit.Increment(clientKey, otpPerClientWindow)
	if err == nil && clientCount > otpPerClientLimit {
		errcode.Abort(ctx, http.StatusTooManyRequests, errcode.OT_TOO_MANY_002, "OTP quota exceeded, contact staff")
		return false
	}
	return true
}

func handleVerifyOtp(ctx *gin.Context, repository *domain.Repository) {
	clientId, id, ok := parseCheckInRef(ctx)
	if !ok {
		return
	}
	var req request.VerifyOtp
	if err := ctx.ShouldBindJSON(&req); err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.OT_BAD_REQUEST_001, err.Error())
		return
	}
	checkIn, err := repository.CheckIn.GetCheckInById(clientId, id)
	if err != nil {
		errcode.Abort(ctx, http.StatusNotFound, errcode.CK_NOT_FOUND_001, "check-in not found")
		return
	}
	if checkIn.OtpVerifiedAt != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.OT_BAD_REQUEST_002, "already verified")
		return
	}
	otpRequest, err := repository.OtpRequest.GetLatestByCheckInId(checkIn.ClientId, checkIn.Id)
	if err != nil {
		errcode.Abort(ctx, http.StatusGone, errcode.OT_GONE_001, "OTP expired, request a new one")
		return
	}
	if time.Now().After(otpRequest.ExpiresAt) {
		errcode.Abort(ctx, http.StatusGone, errcode.OT_GONE_001, "OTP expired, request a new one")
		return
	}
	if otpRequest.AttemptCount >= constant.OtpMaxAttempts {
		errcode.Abort(ctx, http.StatusTooManyRequests, errcode.OT_TOO_MANY_001, "too many wrong attempts, request a new OTP")
		return
	}
	expectedHash := alerting.HashOtp(otpSecret(), checkIn.Phone, otpRequest.RefCode, req.Otp)
	if expectedHash != otpRequest.OtpHash {
		attempts, _ := repository.OtpRequest.IncrementAttempt(checkIn.ClientId, otpRequest.Id)
		errcode.Abort(ctx, http.StatusBadRequest, errcode.OT_BAD_REQUEST_002,
			fmt.Sprintf("invalid OTP, %d attempts remaining", constant.OtpMaxAttempts-attempts))
		return
	}
	completeVerification(ctx, repository, checkIn, otpRequest)
}

func completeVerification(ctx *gin.Context, repository *domain.Repository, checkIn entities.CheckIn, otpRequest entities.OtpRequest) {
	now := time.Now()
	setting, err := repository.BranchSetting.GetSetting(checkIn.ClientId, checkIn.BranchId)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.OT_INTERNAL_001, err.Error())
		return
	}
	rawToken, err := alerting.GenerateSessionToken()
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.OT_INTERNAL_001, err.Error())
		return
	}
	sessionToken := alerting.ComposeTenantRef(checkIn.ClientId, rawToken)
	expiresAt := now.Add(time.Duration(setting.RetentionHours) * time.Hour)
	if err := repository.CheckIn.MarkOtpVerified(checkIn.ClientId, checkIn.Id, now, alerting.HashToken(sessionToken), expiresAt); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.OT_INTERNAL_001, err.Error())
		return
	}
	_ = repository.OtpRequest.MarkVerified(checkIn.ClientId, otpRequest.Id, now)
	response.Ok(ctx, gin.H{
		"sessionToken": sessionToken,
		"status":       "ACTIVE",
		"expiresAt":    expiresAt,
	})
}

func otpSecret() string {
	return os.Getenv("SECRET_KEY")
}
