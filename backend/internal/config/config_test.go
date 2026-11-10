package config

import "testing"

func TestGoogleOAuthReadsStandardEnvNames(t *testing.T) {
	t.Setenv("DATABASE_APP_URL", "postgres://outreach/outreach")
	t.Setenv("MASTER_KEYS", "1:00")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "cid.apps.googleusercontent.com")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "sec")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URI", "https://api.example/callback")
	t.Setenv("PUBLIC_APP_URL", "https://app.example")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GoogleOAuthClientID != "cid.apps.googleusercontent.com" {
		t.Fatalf("client id %q", cfg.GoogleOAuthClientID)
	}
	if cfg.GoogleOAuthClientSecret != "sec" {
		t.Fatalf("secret %q", cfg.GoogleOAuthClientSecret)
	}
	if cfg.GoogleOAuthRedirectURI != "https://api.example/callback" {
		t.Fatalf("redirect %q", cfg.GoogleOAuthRedirectURI)
	}
}
