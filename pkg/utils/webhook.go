package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// SendWebhook sends a JSON POST request to the specified webhookURL with retries and exponential backoff.
func SendWebhook(ctx context.Context, webhookURL string, payload interface{}, maxRetries int, timeout time.Duration) error {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("sendWebhook: marshal payload: %w", err)
	}

	client := &http.Client{
		Timeout: timeout,
	}

	go func() {
		// Create a background context with a timeout per retry attempt (detached from the request context,
		// as the request context may be canceled once the HTTP response is sent back to the user)
		bgCtx := context.Background()

		backoff := 500 * time.Millisecond
		var lastErr error

		for attempt := 1; attempt <= maxRetries; attempt++ {
			req, err := http.NewRequestWithContext(bgCtx, http.MethodPost, webhookURL, bytes.NewBuffer(bodyBytes))
			if err != nil {
				log.Error().Err(err).Msg("sendWebhook: failed to create request")
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "ShortenerAPI-Webhook/1.0")

			log.Debug().
				Str("url", webhookURL).
				Int("attempt", attempt).
				Msg("Sending webhook payload...")

			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					log.Info().
						Str("url", webhookURL).
						Int("status", resp.StatusCode).
						Msg("Webhook delivered successfully")
					return
				}
				lastErr = fmt.Errorf("received status code %d", resp.StatusCode)
			} else {
				lastErr = err
			}

			log.Warn().
				Err(lastErr).
				Str("url", webhookURL).
				Int("attempt", attempt).
				Msg("Webhook delivery attempt failed")

			if attempt < maxRetries {
				select {
				case <-time.After(backoff):
					backoff *= 2
				case <-ctx.Done():
					log.Warn().Msg("Webhook delivery context cancelled during backoff sleep")
					return
				}
			}
		}

		log.Error().
			Err(lastErr).
			Str("url", webhookURL).
			Msg("Failed to deliver webhook after maximum retries")
	}()

	return nil
}
