package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/axiom-nz/corsway/internal/config"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		trustProxy bool
		want       string
	}{
		{
			name: "X-Forwarded-For is present",
			headers: map[string]string{
				"X-Forwarded-For": "1.2.3.4",
			},
			remoteAddr: "5.6.7.8:12345",
			trustProxy: true,
			want:       "1.2.3.4",
		},
		{
			name: "X-Forwarded-For is present but trustProxy disabled",
			headers: map[string]string{
				"X-Forwarded-For": "1.2.3.4",
			},
			remoteAddr: "5.6.7.8:12345",
			trustProxy: false,
			want:       "5.6.7.8",
		},
		{
			name: "Multiple X-Forwarded-For ips",
			headers: map[string]string{
				"X-Forwarded-For": "1.2.3.4, 10.0.0.1",
			},
			remoteAddr: "5.6.7.8:12345",
			trustProxy: true,
			want:       "1.2.3.4",
		},
		{
			name: "X-Forwarded-For with spaces",
			headers: map[string]string{
				"X-Forwarded-For": "  1.2.3.4  , 10.0.0.1",
			},
			remoteAddr: "5.6.7.8:12345",
			trustProxy: true,
			want:       "1.2.3.4",
		},
		{
			name:       "No header, fallback to RemoteAddr with port",
			headers:    map[string]string{},
			remoteAddr: "5.6.7.8:12345",
			trustProxy: true,
			want:       "5.6.7.8",
		},
		{
			name:       "No header, fallback to RemoteAddr without port",
			headers:    map[string]string{},
			remoteAddr: "5.6.7.8",
			trustProxy: true,
			want:       "5.6.7.8",
		},
		{
			name:       "No header, fallback to invalid RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "invalid-address",
			trustProxy: true,
			want:       "invalid-address",
		},
		{
			name:       "Empty case",
			headers:    map[string]string{},
			remoteAddr: "",
			trustProxy: true,
			want:       "",
		},
		{
			name: "TrustProxy false ignores X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "1.2.3.4",
			},
			remoteAddr: "5.6.7.8:12345",
			trustProxy: false,
			want:       "5.6.7.8",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			for k, v := range test.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = test.remoteAddr

			cfg := &config.Config{TrustProxy: test.trustProxy}
			got := getClientIP(cfg, req)
			if got != test.want {
				t.Errorf("getClientIP() = %q, want %q", got, test.want)
			}
		})
	}
}
