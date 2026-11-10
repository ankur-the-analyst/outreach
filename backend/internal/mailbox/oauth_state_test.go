package mailbox

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOAuthStateRoundTrip(t *testing.T) {
	t.Parallel()
	key := []byte("test-key-32-bytes-long-enough!!")
	ws := uuid.MustParse("01700000-0000-7000-8000-000000000001")
	user := uuid.MustParse("01700000-0000-7000-8000-000000000002")
	tok, err := signOAuthState(key, oauthState{WorkspaceID: ws, UserID: user, Timezone: "UTC", Exp: time.Now().Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	got, err := verifyOAuthState(key, tok, time.Now())
	if err != nil || got.WorkspaceID != ws || got.UserID != user || got.Timezone != "UTC" {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := verifyOAuthState([]byte("other"), tok, time.Now()); err == nil {
		t.Fatal("expected bad mac")
	}
}
