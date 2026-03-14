package pull

import (
	"testing"
	"time"
)

func TestClientIsAuthenticated(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name   string
		token  authTokenResponse
		expect bool
	}{
		{
			name:   "missing token",
			token:  authTokenResponse{},
			expect: false,
		},
		{
			name: "missing issued at",
			token: authTokenResponse{
				Token:     "abc",
				ExpiresIn: 60,
			},
			expect: false,
		},
		{
			name: "non positive expiry",
			token: authTokenResponse{
				Token:    "abc",
				IssuedAt: now.Format(time.RFC3339),
			},
			expect: false,
		},
		{
			name: "invalid issued at",
			token: authTokenResponse{
				Token:     "abc",
				ExpiresIn: 60,
				IssuedAt:  "not-a-time",
			},
			expect: false,
		},
		{
			name: "expired token",
			token: authTokenResponse{
				Token:     "abc",
				ExpiresIn: 60,
				IssuedAt:  now.Add(-2 * time.Minute).Format(time.RFC3339),
			},
			expect: false,
		},
		{
			name: "valid token",
			token: authTokenResponse{
				Token:     "abc",
				ExpiresIn: 60,
				IssuedAt:  now.Add(-30 * time.Second).Format(time.RFC3339),
			},
			expect: true,
		},
		{
			name: "token inside refresh buffer",
			token: authTokenResponse{
				Token:     "abc",
				ExpiresIn: 60,
				IssuedAt:  now.Add(-(60*time.Second - authRefreshBuffer + time.Second)).Format(time.RFC3339),
			},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{token: tt.token}

			if got := client.IsAuthenticated(); got != tt.expect {
				t.Fatalf("IsAuthenticated() = %v, want %v", got, tt.expect)
			}
		})
	}
}
