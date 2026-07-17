package request

type UpsertTemplate struct {
	Code             string                       `json:"code" binding:"required"`
	TextTh           string                       `json:"textTh" binding:"required"`
	TextEn           string                       `json:"textEn" binding:"required"`
	ChannelOverrides map[string]TemplateOverrides `json:"channelOverrides"`
	Active           bool                         `json:"active"`
}

type TemplateOverrides struct {
	TextTh string `json:"textTh"`
	TextEn string `json:"textEn"`
}

type UpdateBranchSetting struct {
	ShopName           string `json:"shopName"`
	RetentionHours     int    `json:"retentionHours" binding:"required"`
	CooldownSeconds    int    `json:"cooldownSeconds" binding:"required"`
	ConfirmMethod      string `json:"confirmMethod" binding:"required"`
	SkipOtp            bool   `json:"skipOtp"`
	SmsCreditThreshold int    `json:"smsCreditThreshold"`
	ContactChannel     string `json:"contactChannel"`
}

type SetPin struct {
	Pin           string `json:"pin" binding:"required,len=6,numeric"`
	ConfirmMethod string `json:"confirmMethod" binding:"required"`
}

type UpsertPermission struct {
	UserId            string   `json:"userId" binding:"required"`
	BranchId          string   `json:"branchId" binding:"required"`
	Phone             string   `json:"phone"`
	AllowedEventTypes []string `json:"allowedEventTypes"`
	IsTestRecipient   bool     `json:"isTestRecipient"`
	Active            bool     `json:"active"`
}

type CreateQr struct {
	BranchId string `json:"branchId" binding:"required"`
	TableNo  string `json:"tableNo"`
}

type UpdateMessagingConfig struct {
	SmsEnabled        bool   `json:"smsEnabled"`
	LineEnabled       bool   `json:"lineEnabled"`
	SmsApiUrl         string `json:"smsApiUrl"`
	SmsBalanceUrl     string `json:"smsBalanceUrl"`
	SmsApiKey         string `json:"smsApiKey"`
	SmsApiSecret      string `json:"smsApiSecret"`
	SmsSenderId       string `json:"smsSenderId"`
	SmsWebhookSecret  string `json:"smsWebhookSecret"`
	LineChannelToken  string `json:"lineChannelToken"`
	LineChannelSecret string `json:"lineChannelSecret"`
}
