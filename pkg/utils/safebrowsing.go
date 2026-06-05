package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const safeBrowsingEndpoint = "https://safebrowsing.googleapis.com/v4/threatMatches:find"

type safeBrowsingRequest struct {
	Client    sbClient    `json:"client"`
	ThreatInfo sbThreatInfo `json:"threatInfo"`
}

type sbClient struct {
	ClientID      string `json:"clientId"`
	ClientVersion string `json:"clientVersion"`
}

type sbThreatInfo struct {
	ThreatTypes      []string   `json:"threatTypes"`
	PlatformTypes    []string   `json:"platformTypes"`
	ThreatEntryTypes []string   `json:"threatEntryTypes"`
	ThreatEntries    []sbEntry  `json:"threatEntries"`
}

type sbEntry struct {
	URL string `json:"url"`
}

type safeBrowsingResponse struct {
	Matches []struct {
		ThreatType string `json:"threatType"`
	} `json:"matches"`
}

// IsSafeURL returns true if the URL is safe, false if flagged by Safe Browsing.
// Returns true (safe) if apiKey is empty (feature disabled).
func IsSafeURL(ctx context.Context, apiKey, targetURL string) (bool, error) {
	if apiKey == "" {
		return true, nil // Safe Browsing disabled
	}

	body, err := json.Marshal(safeBrowsingRequest{
		Client: sbClient{ClientID: "shortenerapi", ClientVersion: "1.0"},
		ThreatInfo: sbThreatInfo{
			ThreatTypes:      []string{"MALWARE", "SOCIAL_ENGINEERING", "UNWANTED_SOFTWARE", "POTENTIALLY_HARMFUL_APPLICATION"},
			PlatformTypes:    []string{"ANY_PLATFORM"},
			ThreatEntryTypes: []string{"URL"},
			ThreatEntries:    []sbEntry{{URL: targetURL}},
		},
	})
	if err != nil {
		return true, fmt.Errorf("safeBrowsing: marshal: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost,
		fmt.Sprintf("%s?key=%s", safeBrowsingEndpoint, apiKey),
		bytes.NewReader(body),
	)
	if err != nil {
		return true, fmt.Errorf("safeBrowsing: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// On network error, fail open (allow the URL)
		return true, nil
	}
	defer resp.Body.Close()

	var result safeBrowsingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return true, nil // fail open on decode error
	}

	return len(result.Matches) == 0, nil
}
