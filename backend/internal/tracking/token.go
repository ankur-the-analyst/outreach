package tracking

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	KindOpen  = "open"
	KindClick = "click"
)

type Hasher interface {
	HMAC(msg []byte) []byte
}

type TokenData struct {
	WorkspaceID uuid.UUID
	DraftID     uuid.UUID
	Kind        string
	URL         string
}

func Token(h Hasher, in TokenData) (string, error) {
	if h == nil {
		return "", fmt.Errorf("tracking: hasher required")
	}
	if in.WorkspaceID == uuid.Nil || in.DraftID == uuid.Nil {
		return "", fmt.Errorf("tracking: invalid token")
	}
	if in.Kind != KindOpen && in.Kind != KindClick {
		return "", fmt.Errorf("tracking: invalid token")
	}
	payload := in.WorkspaceID.String() + "\n" + in.DraftID.String() + "\n" + in.Kind + "\n" + in.URL
	mac := hex.EncodeToString(h.HMAC([]byte("track:" + payload)))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + mac, nil
}

func ParseToken(h Hasher, token string) (TokenData, error) {
	if h == nil {
		return TokenData{}, fmt.Errorf("tracking: hasher required")
	}
	token = strings.TrimSpace(token)
	payloadB64, macHex, ok := strings.Cut(token, ".")
	if !ok || payloadB64 == "" || macHex == "" {
		return TokenData{}, fmt.Errorf("tracking: invalid token")
	}
	raw, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return TokenData{}, fmt.Errorf("tracking: invalid token")
	}
	want := h.HMAC([]byte("track:" + string(raw)))
	got, err := hex.DecodeString(macHex)
	if err != nil || !hmac.Equal(want, got) {
		return TokenData{}, fmt.Errorf("tracking: invalid token")
	}
	parts := strings.SplitN(string(raw), "\n", 4)
	if len(parts) != 4 {
		return TokenData{}, fmt.Errorf("tracking: invalid token")
	}
	wsID, err := uuid.Parse(parts[0])
	if err != nil {
		return TokenData{}, fmt.Errorf("tracking: invalid token")
	}
	draftID, err := uuid.Parse(parts[1])
	if err != nil {
		return TokenData{}, fmt.Errorf("tracking: invalid token")
	}
	kind := parts[2]
	if kind != KindOpen && kind != KindClick {
		return TokenData{}, fmt.Errorf("tracking: invalid token")
	}
	return TokenData{WorkspaceID: wsID, DraftID: draftID, Kind: kind, URL: parts[3]}, nil
}
