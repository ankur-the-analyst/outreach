package mailbox

import "testing"

func TestRejectGmailServiceAccount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		email, kind string
		wantErr     error
	}{
		{"user@gmail.com", "service_account", ErrGmailServiceAccount},
		{"User@Googlemail.com", "service_account", ErrGmailServiceAccount},
		{"user@gmail.com", "app_password", nil},
		{"user@acme.com", "service_account", nil},
		{"user@acme.com", "smtp_imap", nil},
	}
	for _, tt := range tests {
		t.Run(tt.email+"/"+tt.kind, func(t *testing.T) {
			t.Parallel()
			err := RejectGmailServiceAccount(tt.email, tt.kind)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("got %v", err)
				}
				return
			}
			if err != tt.wantErr {
				t.Fatalf("got %v want %v", err, tt.wantErr)
			}
		})
	}
}

func TestProviderFor(t *testing.T) {
	t.Parallel()
	if ProviderFor("a@gmail.com") != "gmail" {
		t.Fatal("gmail")
	}
	if ProviderFor("a@acme.com") != "other" {
		t.Fatal("other")
	}
}
