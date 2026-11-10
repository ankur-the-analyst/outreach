package mailbox

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type oauthState struct {
	WorkspaceID uuid.UUID `json:"w"`
	UserID      uuid.UUID `json:"u"`
	Timezone    string    `json:"tz"`
	Exp         int64     `json:"exp"`
}

func signOAuthState(key []byte, s oauthState) (string, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(raw)
	sum := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(sum), nil
}

func verifyOAuthState(key []byte, token string, now time.Time) (oauthState, error) {
	payload, sig, ok := splitDot(token)
	if !ok {
		return oauthState{}, fmt.Errorf("oauth: bad state")
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return oauthState{}, fmt.Errorf("oauth: bad state")
	}
	want, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		return oauthState{}, fmt.Errorf("oauth: bad state")
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(raw)
	if !hmac.Equal(mac.Sum(nil), want) {
		return oauthState{}, fmt.Errorf("oauth: bad state")
	}
	var s oauthState
	if err := json.Unmarshal(raw, &s); err != nil {
		return oauthState{}, fmt.Errorf("oauth: bad state")
	}
	if s.Exp > 0 && now.Unix() > s.Exp {
		return oauthState{}, fmt.Errorf("oauth: state expired")
	}
	return s, nil
}

func splitDot(s string) (string, string, bool) {
	i := 0
	for i < len(s) && s[i] != '.' {
		i++
	}
	if i == 0 || i == len(s)-1 || i >= len(s) {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}
