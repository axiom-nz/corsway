package proxy

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/axiom-nz/corsway/internal/config"
)

func TestProxyIntegration(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Target-Header", "hello")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("target response body"))
	}))
	defer target.Close()

	cfg := &config.Config{
		RateLimit:       100,
		RateLimitWindow: 1 * time.Minute,
		MaxRequestBytes: 1024 * 1024,
	}
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	server := NewServer(cfg, logger)

	testServer := httptest.NewServer(server)
	defer testServer.Close()

	tests := []struct {
		name       string
		targetURL  string
		wantBody   string
		wantStatus int
	}{
		{
			name:       "Standard proxy request",
			targetURL:  target.URL,
			wantBody:   "target response body",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Mangled single slash protocol",
			targetURL:  strings.Replace(target.URL, "://", ":/", 1),
			wantBody:   "target response body",
			wantStatus: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestUrl := testServer.URL + "/" + test.targetURL
			resp, err := http.Get(requestUrl)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != test.wantStatus {
				t.Errorf("got status %d, want %d", resp.StatusCode, test.wantStatus)
			}

			body, _ := io.ReadAll(resp.Body)
			if string(body) != test.wantBody {
				t.Errorf("got body %q, want %q", string(body), test.wantBody)
			}

			// Verify CORS headers were added by the proxy
			if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
				t.Error("missing CORS Access-Control-Allow-Origin header")
			}
		})
	}
}
