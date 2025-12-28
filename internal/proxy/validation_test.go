package proxy

import "testing"

func TestPrepareURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "standard https url",
			input:    "https://store.example.com/saas/list?product_line=abc&store_id=xyz",
			expected: "https://store.example.com/saas/list?product_line=abc&store_id=xyz",
			wantErr:  false,
		},
		{
			name:     "standard http url",
			input:    "http://example.com",
			expected: "http://example.com",
			wantErr:  false,
		},
		{
			name:     "url with whitespace",
			input:    "  https://example.com  ",
			expected: "https://example.com",
			wantErr:  false,
		},
		{
			name:     "protocol-relative url",
			input:    "//example.com/path",
			expected: "https://example.com/path",
			wantErr:  false,
		},
		{
			name:     "missing protocol",
			input:    "example.com/api",
			expected: "https://example.com/api",
			wantErr:  false,
		},
		{
			name:     "url with port",
			input:    "localhost:8080/data",
			expected: "https://localhost:8080/data",
			wantErr:  false,
		},
		{
			name:     "single slash protocol",
			input:    "https:/example.com/api",
			expected: "https://example.com/api",
			wantErr:  false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url, err := PrepareURL(test.input)
			if (err != nil) != test.wantErr {
				t.Errorf("PrepareURL() url, %v, error = %v, wantErr %v", url, err, test.wantErr)
				return
			}
			if !test.wantErr && url != test.expected {
				t.Errorf("PrepareURL() url = %v, want %v", url, test.expected)
			}
		})
	}
}
