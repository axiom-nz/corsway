package proxy

import (
	"fmt"
	"net/url"
	"strings"
)

// PrepareURL normalises and validates a URL
func PrepareURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)

	// Prepend protocol to URL if missing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		if strings.HasPrefix(rawURL, "//") {
			rawURL = "https:" + rawURL
		} else {
			rawURL = "https://" + rawURL
		}
	}

	// Parse and validate the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	// Ensure the URL has a host
	if parsedURL.Host == "" {
		return "", fmt.Errorf("invalid URL: missing host")
	}

	return parsedURL.String(), nil
}
