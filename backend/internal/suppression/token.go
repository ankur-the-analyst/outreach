package suppression

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

func Token(h Hasher, workspaceID uuid.UUID, email string) (string, error) {
	if h == nil {
		return "", fmt.Errorf("suppression: hasher required")
	}
	email = Normalize(email)
	if email == "" {
		return "", fmt.Errorf("suppression: value required")
	}
	payload := workspaceID.String() + "\n" + email
	mac := hex.EncodeToString(h.HMAC([]byte("unsub:" + payload)))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + mac, nil
}

func ParseToken(h Hasher, token string) (uuid.UUID, string, error) {
	if h == nil {
		return uuid.UUID{}, "", fmt.Errorf("suppression: hasher required")
	}
	token = strings.TrimSpace(token)
	payloadB64, macHex, ok := strings.Cut(token, ".")
	if !ok || payloadB64 == "" || macHex == "" {
		return uuid.UUID{}, "", fmt.Errorf("suppression: invalid token")
	}
	raw, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return uuid.UUID{}, "", fmt.Errorf("suppression: invalid token")
	}
	want := h.HMAC([]byte("unsub:" + string(raw)))
	got, err := hex.DecodeString(macHex)
	if err != nil || !hmac.Equal(want, got) {
		return uuid.UUID{}, "", fmt.Errorf("suppression: invalid token")
	}
	wsRaw, email, ok := strings.Cut(string(raw), "\n")
	if !ok {
		return uuid.UUID{}, "", fmt.Errorf("suppression: invalid token")
	}
	wsID, err := uuid.Parse(wsRaw)
	if err != nil {
		return uuid.UUID{}, "", fmt.Errorf("suppression: invalid token")
	}
	email = Normalize(email)
	if email == "" {
		return uuid.UUID{}, "", fmt.Errorf("suppression: invalid token")
	}
	return wsID, email, nil
}

func OneClickURL(base string, h Hasher, workspaceID uuid.UUID, email string) (string, error) {
	tok, err := Token(h, workspaceID, email)
	if err != nil {
		return "", err
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "", fmt.Errorf("suppression: public url required")
	}
	u, err := url.Parse(base + "/api/v1/unsubscribe/" + tok)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
