package request

type TriggerAlert struct {
	EventType        string `json:"eventType" binding:"required"`
	ConfirmMethod    string `json:"confirmMethod" binding:"required"`
	Pin              string `json:"pin"`
	OverrideCooldown bool   `json:"overrideCooldown"`
}
