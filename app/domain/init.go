package domain

import (
	"alert/app/core/messaging"
	"alert/app/data/repositories"
	"alert/db"
)

type Repository struct {
	Session         repositories.ISession
	CheckIn         repositories.ICheckIn
	OtpRequest      repositories.IOtpRequest
	EmergencyEvent  repositories.IEmergencyEvent
	MessageTemplate repositories.IMessageTemplate
	DeliveryLog     repositories.IDeliveryLog
	AuditLog        repositories.IAuditLog
	BranchSetting   repositories.IBranchSetting
	QrToken         repositories.IQrToken
	StaffPermission repositories.IStaffPermission
	Counter         repositories.ICounter
	RateLimit       repositories.IRateLimit
	Dispatcher      *messaging.Dispatcher
	OtpSender       messaging.OtpSender
	SmsBalance      messaging.BalanceChecker
}

func InitRepository(resource *db.Resource) *Repository {
	smsProvider := messaging.NewSmsProvider()
	balanceChecker, _ := smsProvider.(messaging.BalanceChecker)
	return &Repository{
		Session:         repositories.NewSessionEntity(resource),
		CheckIn:         repositories.NewCheckInEntity(resource),
		OtpRequest:      repositories.NewOtpRequestEntity(resource),
		EmergencyEvent:  repositories.NewEmergencyEventEntity(resource),
		MessageTemplate: repositories.NewMessageTemplateEntity(resource),
		DeliveryLog:     repositories.NewDeliveryLogEntity(resource),
		AuditLog:        repositories.NewAuditLogEntity(resource),
		BranchSetting:   repositories.NewBranchSettingEntity(resource),
		QrToken:         repositories.NewQrTokenEntity(resource),
		StaffPermission: repositories.NewStaffPermissionEntity(resource),
		Counter:         repositories.NewCounterEntity(resource),
		RateLimit:       repositories.NewRateLimitEntity(resource),
		Dispatcher:      messaging.NewDispatcher(smsProvider, messaging.NewPushProvider(), messaging.NewLineProvider()),
		OtpSender:       messaging.NewSmsOtpSender(smsProvider),
		SmsBalance:      balanceChecker,
	}
}
