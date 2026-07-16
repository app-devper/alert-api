package alerting

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

const refCodeCharset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

func GenerateOtp() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func GenerateRefCode() (string, error) {
	code := make([]byte, 4)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(refCodeCharset))))
		if err != nil {
			return "", err
		}
		code[i] = refCodeCharset[n.Int64()]
	}
	return string(code), nil
}

func GenerateSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func HashOtp(secret string, phone string, refCode string, otp string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(phone + ":" + refCode + ":" + otp))
	return hex.EncodeToString(mac.Sum(nil))
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func FormatEventNo(prefix string, date time.Time, sequence int64) string {
	return fmt.Sprintf("%s%s%03d", prefix, date.Format("060102"), sequence)
}

func FormatCheckInNo(date time.Time, sequence int64) string {
	return fmt.Sprintf("CI%s%04d", date.Format("060102"), sequence)
}
