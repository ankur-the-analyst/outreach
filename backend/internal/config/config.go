package config

import (
	"crypto/sha256"
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Addr                    string `default:":8080"`
	DatabaseAppURL          string `required:"true" split_words:"true"`
	MasterKeys              string `required:"true" split_words:"true"`
	MasterKeyActive         int    `split_words:"true" default:"1"`
	UndoHoldS               int    `split_words:"true" default:"300"`
	UnknownTimeoutS         int    `split_words:"true" default:"900"`
	MailpitAPI              string `split_words:"true" default:"http://127.0.0.1:8025"`
	PublicBaseURL           string `split_words:"true" default:"http://127.0.0.1:8080"`
	GoogleOAuthClientID     string `split_words:"true"`
	GoogleOAuthClientSecret string `split_words:"true"`
	GoogleOAuthRedirectURI  string `split_words:"true" default:"http://localhost:3000/api/v1/mailboxes/oauth/callback"`
}

func Load() (Config, error) {
	loadDotenv()
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

func (c Config) OAuthStateKey() []byte {
	sum := sha256.Sum256([]byte(c.MasterKeys))
	return sum[:]
}
