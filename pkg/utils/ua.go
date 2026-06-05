package utils

import "strings"

// ParseDeviceType returns "mobile", "tablet", or "desktop" based on User-Agent.
func ParseDeviceType(ua string) string {
	ua = strings.ToLower(ua)
	if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		return "tablet"
	}
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		return "mobile"
	}
	return "desktop"
}

// ParseBrowser returns a human-readable browser name from User-Agent.
func ParseBrowser(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "edg"):
		return "Edge"
	case strings.Contains(ua, "chrome"):
		return "Chrome"
	case strings.Contains(ua, "firefox"):
		return "Firefox"
	case strings.Contains(ua, "safari"):
		return "Safari"
	case strings.Contains(ua, "opera"):
		return "Opera"
	default:
		return "Other"
	}
}

// ParseOS returns the operating system name from User-Agent.
func ParseOS(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		return "iOS"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "mac os"):
		return "macOS"
	case strings.Contains(ua, "linux"):
		return "Linux"
	default:
		return "Other"
	}
}

// ClassifyReferrer returns "direct", "social", "search", "email", or "other".
func ClassifyReferrer(referrer string) string {
	if referrer == "" {
		return "direct"
	}
	ref := strings.ToLower(referrer)
	for _, s := range []string{"facebook", "twitter", "instagram", "linkedin", "tiktok", "youtube", "pinterest"} {
		if strings.Contains(ref, s) {
			return "social"
		}
	}
	for _, s := range []string{"google", "bing", "yahoo", "duckduckgo", "baidu"} {
		if strings.Contains(ref, s) {
			return "search"
		}
	}
	if strings.Contains(ref, "mailto:") {
		return "email"
	}
	return "other"
}
