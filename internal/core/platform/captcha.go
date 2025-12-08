package platform

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type TurnstileResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

func VerifyCaptcha(token string) error {
	// Development bypass - set BYPASS_CAPTCHA=true in .env to skip verification
	if os.Getenv("BYPASS_CAPTCHA") == "true" {
		fmt.Println("[DEBUG] Captcha verification bypassed")
		return nil
	}

	secretKey := os.Getenv("TURNSTILE_SECRET_KEY")
	if secretKey == "" {
		// If no secret key is configured, we might want to skip verification (dev mode)
		// or fail secure. For now, let's log and fail to ensure it's set up.
		return fmt.Errorf("TURNSTILE_SECRET_KEY is not set")
	}

	// If token is empty, fail
	if token == "" {
		return fmt.Errorf("captcha token is empty")
	}

	formData := url.Values{}
	formData.Set("secret", secretKey)
	formData.Set("response", token)

	resp, err := http.PostForm(turnstileVerifyURL, formData)
	if err != nil {
		return fmt.Errorf("failed to verify captcha: %v", err)
	}
	defer resp.Body.Close()

	var result TurnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode captcha response: %v", err)
	}

	if !result.Success {
		return fmt.Errorf("captcha verification failed: %v", result.ErrorCodes)
	}

	return nil
}
