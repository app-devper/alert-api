package request

type CreateCheckIn struct {
	QrToken              string `json:"qrToken"`
	ClientId             string `json:"clientId"`
	BranchId             string `json:"branchId"`
	Phone                string `json:"phone" binding:"required"`
	GroupSize            int    `json:"groupSize" binding:"required,min=1,max=100"`
	TableNo              string `json:"tableNo"`
	PreferredLanguage    string `json:"preferredLanguage"`
	AcceptPrivacyNotice  bool   `json:"acceptPrivacyNotice" binding:"required"`
	PrivacyNoticeVersion string `json:"privacyNoticeVersion" binding:"required"`
	MarketingConsent     bool   `json:"marketingConsent"`
}

type VerifyOtp struct {
	Otp string `json:"otp" binding:"required,len=6"`
}

type PushSubscribe struct {
	Endpoint string `json:"endpoint" binding:"required"`
	P256dh   string `json:"p256dh" binding:"required"`
	Auth     string `json:"auth" binding:"required"`
}
