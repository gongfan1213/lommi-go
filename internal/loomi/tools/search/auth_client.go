package search

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"time"
)

// AuthClient handles authentication for social media search API
type AuthClient struct {
	clientID  string
	secretKey string
	baseURL   string
}

// NewAuthClient creates a new authentication client
func NewAuthClient(clientID, secretKey string) *AuthClient {
	return &AuthClient{
		clientID:  clientID,
		secretKey: secretKey,
	}
}

// GetHeaders generates authentication headers for API requests
func (ac *AuthClient) GetHeaders() map[string]string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Generate signature using MD5
	signature := ac.generateSignature(timestamp)

	return map[string]string{
		"Content-Type": "application/json",
		"X-Client-ID":  ac.clientID,
		"X-Timestamp":  timestamp,
		"X-Signature":  signature,
	}
}

// generateSignature generates MD5 signature for authentication
func (ac *AuthClient) generateSignature(timestamp string) string {
	// Combine client_id, secret_key, and timestamp
	data := ac.clientID + ac.secretKey + timestamp

	// Generate MD5 hash
	hash := md5.Sum([]byte(data))

	// Convert to hex string
	return fmt.Sprintf("%x", hash)
}
