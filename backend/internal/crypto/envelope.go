package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	dekSize   = 32
	nonceSize = 12
)

var (
	ErrUnknownKeyVersion = errors.New("crypto: unknown key version")
	ErrInvalidMasterKeys = errors.New("crypto: invalid MASTER_KEYS")
	ErrOpen              = errors.New("crypto: open failed")
)

type Sealed struct {
	KeyVersion int
	WrappedDEK []byte
	Nonce      []byte
	Ciphertext []byte
}

type Keyring struct {
	active int
	keys   map[int][]byte
}

func NewKeyring(spec string, active int) (*Keyring, error) {
	keys := make(map[int][]byte)
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("%w: empty", ErrInvalidMasterKeys)
	}
	for _, part := range strings.Split(spec, ",") {
		ver, key, err := parseKeyPart(part)
		if err != nil {
			return nil, err
		}
		keys[ver] = key
	}
	if _, ok := keys[active]; !ok {
		return nil, fmt.Errorf("%w: active version %d missing", ErrInvalidMasterKeys, active)
	}
	return &Keyring{active: active, keys: keys}, nil
}

func parseKeyPart(part string) (int, []byte, error) {
	verStr, hexKey, ok := strings.Cut(strings.TrimSpace(part), ":")
	if !ok {
		return 0, nil, fmt.Errorf("%w: want version:hex", ErrInvalidMasterKeys)
	}
	ver, err := strconv.Atoi(verStr)
	if err != nil || ver < 1 {
		return 0, nil, fmt.Errorf("%w: version", ErrInvalidMasterKeys)
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != dekSize {
		return 0, nil, fmt.Errorf("%w: key must be 32 bytes hex", ErrInvalidMasterKeys)
	}
	return ver, key, nil
}

func (k *Keyring) Seal(plain []byte) (Sealed, error) {
	kek, ok := k.keys[k.active]
	if !ok {
		return Sealed{}, ErrUnknownKeyVersion
	}
	dek := make([]byte, dekSize)
	if _, err := rand.Read(dek); err != nil {
		return Sealed{}, fmt.Errorf("crypto: dek: %w", err)
	}
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return Sealed{}, fmt.Errorf("crypto: nonce: %w", err)
	}
	ct, err := gcmSeal(dek, nonce, plain)
	if err != nil {
		return Sealed{}, err
	}
	wrapped, err := wrapDEK(kek, dek)
	if err != nil {
		return Sealed{}, err
	}
	return Sealed{
		KeyVersion: k.active,
		WrappedDEK: wrapped,
		Nonce:      nonce,
		Ciphertext: ct,
	}, nil
}

func (k *Keyring) HMAC(msg []byte) []byte {
	mac := hmac.New(sha256.New, k.keys[k.active])
	mac.Write(msg)
	return mac.Sum(nil)
}

func (k *Keyring) Open(s Sealed) ([]byte, error) {
	kek, ok := k.keys[s.KeyVersion]
	if !ok {
		return nil, ErrUnknownKeyVersion
	}
	dek, err := unwrapDEK(kek, s.WrappedDEK)
	if err != nil {
		return nil, err
	}
	plain, err := gcmOpen(dek, s.Nonce, s.Ciphertext)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

func wrapDEK(kek, dek []byte) ([]byte, error) {
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: wrap nonce: %w", err)
	}
	ct, err := gcmSeal(kek, nonce, dek)
	if err != nil {
		return nil, err
	}
	return append(nonce, ct...), nil
}

func unwrapDEK(kek, wrapped []byte) ([]byte, error) {
	if len(wrapped) < nonceSize+1 {
		return nil, ErrOpen
	}
	return gcmOpen(kek, wrapped[:nonceSize], wrapped[nonceSize:])
}

func gcmSeal(key, nonce, plain []byte) ([]byte, error) {
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(nonce) != aead.NonceSize() {
		return nil, ErrOpen
	}
	return aead.Seal(nil, nonce, plain, nil), nil
}

func gcmOpen(key, nonce, ct []byte) ([]byte, error) {
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	plain, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, ErrOpen
	}
	return plain, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm: %w", err)
	}
	return aead, nil
}
