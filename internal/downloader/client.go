package downloader

import (
	"net/http"
	"time"
)

// NewClient creates a configured HTTP client with proper timeouts.
func NewClient(userAgent string) *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
