package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	CallbackSignatureHeader = "X-Callback-Signature"
	CallbackTimestampHeader = "X-Callback-Timestamp"
)

func BuildCallbackSignature(secret, timestamp, runID, runToken, correlationID string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	mac.Write([]byte(strings.TrimSpace(timestamp)))
	mac.Write([]byte{0})
	mac.Write([]byte(strings.TrimSpace(runID)))
	mac.Write([]byte{0})
	mac.Write([]byte(strings.TrimSpace(runToken)))
	mac.Write([]byte{0})
	mac.Write([]byte(strings.TrimSpace(correlationID)))
	mac.Write([]byte{0})
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyCallbackSignature(secret, providedSignature, timestamp, runID, runToken, correlationID string, body []byte, now time.Time, maxSkew time.Duration) error {
	provided := strings.TrimSpace(strings.ToLower(providedSignature))
	if provided == "" {
		return errors.New("missing callback signature")
	}

	tsRaw := strings.TrimSpace(timestamp)
	if tsRaw == "" {
		return errors.New("missing callback timestamp")
	}
	ts, err := strconv.ParseInt(tsRaw, 10, 64)
	if err != nil {
		return errors.New("invalid callback timestamp")
	}

	delta := now.Unix() - ts
	if delta < 0 {
		delta = -delta
	}
	if time.Duration(delta)*time.Second > maxSkew {
		return errors.New("callback timestamp outside allowed window")
	}

	expected := BuildCallbackSignature(secret, tsRaw, runID, runToken, correlationID, body)
	if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
		return errors.New("invalid callback signature")
	}
	return nil
}
