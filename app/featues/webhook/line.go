package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"alert/app/core/alerting"
	"alert/app/core/constant"
	"alert/app/core/errcode"
	"alert/app/core/response"
	"alert/app/domain"

	"github.com/gin-gonic/gin"
)

type lineWebhookPayload struct {
	Events []struct {
		Type     string `json:"type"`
		Delivery struct {
			Data string `json:"data"`
		} `json:"delivery"`
	} `json:"events"`
}

func handleLineWebhook(ctx *gin.Context, repository *domain.Repository) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.WH_BAD_REQUEST_001, "unreadable body")
		return
	}
	if !VerifyLineSignature(os.Getenv("LINE_CHANNEL_SECRET"), body, ctx.GetHeader("X-Line-Signature")) {
		errcode.Abort(ctx, http.StatusUnauthorized, errcode.WH_UNAUTHORIZED_001, "invalid signature")
		return
	}
	var payload lineWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.WH_BAD_REQUEST_001, "invalid payload")
		return
	}
	now := time.Now()
	updated := 0
	for _, event := range payload.Events {
		if event.Type != "delivery" || event.Delivery.Data == "" {
			continue
		}
		clientId, _, err := alerting.SplitTenantRef(event.Delivery.Data)
		if err != nil {
			continue
		}
		if _, err := repository.DeliveryLog.UpdateStatusByProviderReference(clientId, event.Delivery.Data, constant.DeliveryDelivered, "delivery", now); err == nil {
			updated++
		}
	}
	response.Ok(ctx, gin.H{"updated": updated})
}

func VerifyLineSignature(channelSecret string, body []byte, signature string) bool {
	if channelSecret == "" {
		return true
	}
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
