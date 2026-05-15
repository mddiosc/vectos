package content

import (
	"testing"
)

func TestShouldSkipFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantSkip bool
	}{
		// .env variants — should skip
		{".env", ".env", true},
		{".env.local", ".env.local", true},
		{".env.production", ".env.production", true},
		{".env.development", ".env.development", true},
		// SSH private keys — should skip
		{"id_rsa", "id_rsa", true},
		{"id_ecdsa", "id_ecdsa", true},
		{"id_ed25519", "id_ed25519", true},
		{"custom_rsa", "custom_rsa", true},
		{"prod_ecdsa", "prod_ecdsa", true},
		{"deploy_ed25519", "deploy_ed25519", true},
		// Certificate/credential files — should skip
		{"cert.pem", "cert.pem", true},
		{"private.key", "private.key", true},
		{"keystore.pfx", "keystore.pfx", true},
		{"keystore.p12", "keystore.p12", true},
		{"credentials.json", "credentials.json", true},
		{"service-account.json", "service-account.json", true},
		// Non-sensitive — should NOT skip
		{".env.example", ".env.example", false},
		{".env.sample", ".env.sample", false},
		{"main.go", "main.go", false},
		{"app.ts", "app.ts", false},
		{"Dockerfile", "Dockerfile", false},
		{"README.md", "README.md", false},
		{"config.yaml", "config.yaml", false},
		{"id_rsa.pub", "id_rsa.pub", false},
		// Edge cases
		{"empty name", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSkipFile(tt.filename)
			if got != tt.wantSkip {
				t.Errorf("ShouldSkipFile(%q) = %v, want %v", tt.filename, got, tt.wantSkip)
			}
		})
	}
}
