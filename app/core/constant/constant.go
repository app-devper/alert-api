package constant

const (
	SUPER   = "SUPER"
	ADMIN   = "ADMIN"
	MANAGER = "MANAGER"
	STAFF   = "STAFF"
)

const (
	EventFire             = "FIRE"
	EventEvacuate         = "EVACUATE"
	EventAvoidArea        = "AVOID_AREA"
	EventSuspiciousObject = "SUSPICIOUS_OBJECT"
	EventBrawl            = "BRAWL"
	EventExternal         = "EXTERNAL"
	EventAllClear         = "ALL_CLEAR"
	EventTest             = "TEST"
)

var EventTypes = []string{
	EventFire, EventEvacuate, EventAvoidArea, EventSuspiciousObject,
	EventBrawl, EventExternal, EventAllClear, EventTest,
}

func IsValidEventType(eventType string) bool {
	for _, t := range EventTypes {
		if t == eventType {
			return true
		}
	}
	return false
}

const (
	ChannelSms  = "SMS"
	ChannelPush = "PUSH"
	ChannelLine = "LINE"
)

const (
	DeliveryQueued    = "QUEUED"
	DeliverySent      = "SENT"
	DeliveryDelivered = "DELIVERED"
	DeliveryFailed    = "FAILED"
)

const (
	ConfirmHold3s = "HOLD_3S"
	ConfirmPin    = "PIN"
)

const (
	CheckedOutBySelf    = "SELF"
	CheckedOutByExpired = "EXPIRED"
)

const (
	LanguageTh = "TH"
	LanguageEn = "EN"
)

const (
	EventStatusOpen   = "OPEN"
	EventStatusClosed = "CLOSED"
)

const (
	ActionTriggerAlert     = "TRIGGER_ALERT"
	ActionTestAlert        = "TEST_ALERT"
	ActionCheckoutCustomer = "CHECKOUT_CUSTOMER"
	ActionUpdateTemplate   = "UPDATE_TEMPLATE"
	ActionChangePin        = "CHANGE_PIN"
	ActionViewCheckinList  = "VIEW_CHECKIN_LIST"
	ActionExportReport     = "EXPORT_REPORT"
	ActionWithdrawConsent  = "WITHDRAW_CONSENT"
	ActionUpdateSetting    = "UPDATE_SETTING"
	ActionUpdatePermission = "UPDATE_PERMISSION"
	ActionManageQr         = "MANAGE_QR"
	ActionCooldownOverride = "COOLDOWN_OVERRIDE"
)

const (
	ResultSuccess = "SUCCESS"
	ResultFailed  = "FAILED"
)

const (
	DefaultRetentionHours  = 24
	MinRetentionHours      = 6
	MaxRetentionHours      = 24
	DefaultCooldownSeconds = 180
	MinCooldownSeconds     = 60
	MaxCooldownSeconds     = 180
	OtpExpiryMinutes       = 5
	OtpMaxAttempts         = 5
	PinMaxAttempts         = 5
	PinLockMinutes         = 10
)
