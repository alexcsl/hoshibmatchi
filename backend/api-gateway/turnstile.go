package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// TurnstileVerificationResponse represents the response from Cloudflare Turnstile API
type TurnstileVerificationResponse struct {
	Success     bool     `json:"success"`
	ChallengeTs string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
	Action      string   `json:"action"`
	CData       string   `json:"cdata"`
}

// VerifyTurnstileToken verifies a Cloudflare Turnstile token
func VerifyTurnstileToken(token string, remoteIP string) (bool, error) {
	secretKey := os.Getenv("VITE_TURNSTILE_SECRET_KEY")
	if secretKey == "" {
		return false, fmt.Errorf("Turnstile secret key not configured")
	}

	// Prepare the verification request
	reqBody := map[string]string{
		"secret":   secretKey,
		"response": token,
		"remoteip": remoteIP,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Make request to Cloudflare
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Post(
		"https://challenges.cloudflare.com/turnstile/v0/siteverify",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return false, fmt.Errorf("failed to verify turnstile: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %v", err)
	}

	var verifyResp TurnstileVerificationResponse
	if err := json.Unmarshal(body, &verifyResp); err != nil {
		return false, fmt.Errorf("failed to parse response: %v", err)
	}

	if !verifyResp.Success {
		return false, fmt.Errorf("verification failed: %v", verifyResp.ErrorCodes)
	}

	return true, nil
}
