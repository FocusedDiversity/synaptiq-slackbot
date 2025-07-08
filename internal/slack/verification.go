package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// RequestVerifier verifies Slack request signatures
type RequestVerifier struct {
	signingSecret string
}

// NewRequestVerifier creates a new request verifier
func NewRequestVerifier(signingSecret string) *RequestVerifier {
	return &RequestVerifier{
		signingSecret: signingSecret,
	}
}

// VerifyRequest verifies a Slack request signature
func (v *RequestVerifier) VerifyRequest(timestamp, signature, body string) error {
	// Check timestamp to prevent replay attacks
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	if time.Now().Unix()-ts > 60*5 {
		return fmt.Errorf("request timestamp too old")
	}

	// Verify signature
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, body)

	h := hmac.New(sha256.New, []byte(v.signingSecret))
	h.Write([]byte(baseString))
	computedSignature := "v0=" + hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(computedSignature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
