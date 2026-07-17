package messaging

import (
	"errors"
	"fmt"
)

type OtpSender interface {
	SendOtp(cfg ProviderConfig, phone string, refCode string, otp string) error
}

type smsOtpSender struct {
	provider MessageProvider
}

func NewSmsOtpSender(provider MessageProvider) OtpSender {
	return &smsOtpSender{provider: provider}
}

func (s *smsOtpSender) SendOtp(cfg ProviderConfig, phone string, refCode string, otp string) error {
	text := fmt.Sprintf("รหัสยืนยัน %s (Ref: %s) หมดอายุใน 5 นาที ห้ามบอกรหัสนี้แก่ผู้อื่น", otp, refCode)
	results := s.provider.Send(cfg, []OutboundMessage{{RecipientKey: "otp", Target: phone, Text: text}})
	if len(results) == 0 {
		return errors.New("otp send produced no result")
	}
	if !results[0].Success {
		return errors.New(results[0].FailReason)
	}
	return nil
}
