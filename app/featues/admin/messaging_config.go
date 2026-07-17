package admin

import (
	"net/http"

	"alert/app/core/constant"
	"alert/app/core/errcode"
	"alert/app/core/response"
	"alert/app/data/entities"
	"alert/app/domain"
	"alert/app/domain/request"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func handleGetMessagingConfig(ctx *gin.Context, repository *domain.Repository) {
	config, err := repository.MessagingConfig.GetConfig(ctx.GetString("ClientId"))
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
		return
	}
	response.Ok(ctx, maskedMessagingConfig(config))
}

func maskedMessagingConfig(config entities.MessagingConfig) gin.H {
	return gin.H{
		"smsEnabled":           config.SmsEnabled,
		"lineEnabled":          config.LineEnabled,
		"smsApiUrl":            config.SmsApiUrl,
		"smsBalanceUrl":        config.SmsBalanceUrl,
		"smsSenderId":          config.SmsSenderId,
		"smsApiKeyMasked":      maskTail(config.SmsApiKey),
		"hasSmsApiSecret":      config.SmsApiSecret != "",
		"hasSmsWebhookSecret":  config.SmsWebhookSecret != "",
		"lineTokenMasked":      maskTail(config.LineChannelToken),
		"hasLineChannelSecret": config.LineChannelSecret != "",
		"updatedBy":            config.UpdatedBy,
		"updatedAt":            config.UpdatedAt,
	}
}

func maskTail(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 4 {
		return "••••"
	}
	return "••••" + secret[len(secret)-4:]
}

func handleUpdateMessagingConfig(ctx *gin.Context, repository *domain.Repository) {
	var req request.UpdateMessagingConfig
	if err := ctx.ShouldBindJSON(&req); err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_001, err.Error())
		return
	}
	clientId := ctx.GetString("ClientId")
	existing, err := repository.MessagingConfig.GetConfig(clientId)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
		return
	}
	config := entities.MessagingConfig{
		ClientId:          clientId,
		SmsEnabled:        req.SmsEnabled,
		LineEnabled:       req.LineEnabled,
		SmsApiUrl:         req.SmsApiUrl,
		SmsBalanceUrl:     req.SmsBalanceUrl,
		SmsSenderId:       req.SmsSenderId,
		SmsApiKey:         keepIfBlank(req.SmsApiKey, existing.SmsApiKey),
		SmsApiSecret:      keepIfBlank(req.SmsApiSecret, existing.SmsApiSecret),
		SmsWebhookSecret:  keepIfBlank(req.SmsWebhookSecret, existing.SmsWebhookSecret),
		LineChannelToken:  keepIfBlank(req.LineChannelToken, existing.LineChannelToken),
		LineChannelSecret: keepIfBlank(req.LineChannelSecret, existing.LineChannelSecret),
		UpdatedBy:         ctx.GetString("UserId"),
	}
	if err := repository.MessagingConfig.UpsertConfig(config); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
		return
	}
	auditAdmin(ctx, repository, constant.ActionUpdateSetting, bson.M{
		"target":      "MESSAGING_CONFIG",
		"smsEnabled":  req.SmsEnabled,
		"lineEnabled": req.LineEnabled,
		"smsSenderId": req.SmsSenderId,
	})
	response.Ok(ctx, maskedMessagingConfig(config))
}

func keepIfBlank(incoming string, existing string) string {
	if incoming == "" {
		return existing
	}
	return incoming
}
